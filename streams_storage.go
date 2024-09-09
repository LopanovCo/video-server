package videoserver

import (
	"fmt"
	"sync"

	"github.com/LdDl/video-server/configuration"
	"github.com/LdDl/video-server/storage"
	"github.com/deepch/vdk/av"
	"github.com/deepch/vdk/codec/aacparser"
	"github.com/deepch/vdk/codec/h264parser"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

const (
	SCOPE_STREAM = "stream"
)

// StreamsStorage Map wrapper for map[uuid.UUID]*StreamConfiguration with mutex for concurrent usage
type StreamsStorage struct {
	sync.RWMutex
	Streams map[uuid.UUID]*StreamConfiguration `json:"rtsp_streams"`

	minioStorage storage.ArchiveStorage
}

// NewStreamsStorageDefault prepares new allocated storage
func NewStreamsStorageDefault() StreamsStorage {
	return StreamsStorage{Streams: make(map[uuid.UUID]*StreamConfiguration)}
}

func (sm *StreamsStorage) GetStream(id uuid.UUID) (string, []StreamType) {
	sm.Lock()
	defer sm.Unlock()
	return sm.Streams[id].URL, sm.Streams[id].SupportedOutputTypes
}

// getKeys returns all storage streams' keys as slice
func (sm *StreamsStorage) getKeys() []uuid.UUID {
	sm.Lock()
	keys := make([]uuid.UUID, 0, len(sm.Streams))
	for k := range sm.Streams {
		keys = append(keys, k)
	}
	sm.Unlock()
	return keys
}

func (streams *StreamsStorage) archiveEnabled(streamID uuid.UUID) (bool, error) {
	streams.RLock()
	defer streams.RUnlock()
	stream, ok := streams.Streams[streamID]
	if !ok {
		return false, ErrStreamNotFound
	}
	return stream.archive != nil, nil
}

func (streams *StreamsStorage) streamExists(streamID uuid.UUID) bool {
	streams.RLock()
	_, ok := streams.Streams[streamID]
	streams.RUnlock()
	return ok
}

func (streams *StreamsStorage) existsWithType(streamID uuid.UUID, streamType StreamType) bool {
	streams.Lock()
	defer streams.Unlock()
	stream, ok := streams.Streams[streamID]
	if !ok {
		return false
	}
	supportedTypes := stream.SupportedOutputTypes
	typeEnabled := typeExists(streamType, supportedTypes)
	return ok && typeEnabled
}

func (streams *StreamsStorage) addCodec(streamID uuid.UUID, codecs []av.CodecData) {
	streams.Lock()
	defer streams.Unlock()
	stream, ok := streams.Streams[streamID]
	if !ok {
		return
	}
	stream.Codecs = codecs
	if stream.verboseLevel > VERBOSE_SIMPLE {
		log.Info().Str("scope", SCOPE_STREAM).Str("event", "codec_add").Str("stream_id", streamID.String()).Any("codec_data", codecs).Msg("Add codec")
	}
}

func (streams *StreamsStorage) getCodec(streamID uuid.UUID) ([]av.CodecData, error) {
	streams.Lock()
	defer streams.Unlock()
	stream, ok := streams.Streams[streamID]
	if !ok {
		return nil, ErrStreamNotFound
	}
	codecs := make([]av.CodecData, len(stream.Codecs))
	for i, iface := range stream.Codecs {
		switch codecType := iface.(type) {
		case aacparser.CodecData, h264parser.CodecData:
			codecs[i] = codecType
		default:
			return nil, fmt.Errorf("unknown codec type: %T", iface)
		}
	}
	return codecs, nil
}

func (streams *StreamsStorage) updateStreamStatus(streamID uuid.UUID, status bool) error {
	streams.Lock()
	defer streams.Unlock()
	stream, ok := streams.Streams[streamID]
	if !ok {
		return ErrStreamNotFound
	}
	stream.Status = status
	if stream.verboseLevel > VERBOSE_SIMPLE {
		log.Info().Str("scope", SCOPE_STREAM).Str("event", "status_update").Str("stream_id", streamID.String()).Bool("status", status).Msg("Status update")
	}
	return nil
}

func (streams *StreamsStorage) addClient(streamID uuid.UUID) (uuid.UUID, chan av.Packet, error) {
	streams.Lock()
	defer streams.Unlock()
	stream, ok := streams.Streams[streamID]
	if !ok {
		return uuid.UUID{}, nil, ErrStreamNotFound
	}
	clientID, err := uuid.NewUUID()
	if err != nil {
		return uuid.UUID{}, nil, err
	}
	if stream.verboseLevel > VERBOSE_SIMPLE {
		log.Info().Str("scope", SCOPE_STREAM).Str("event", "client_add").Str("stream_id", streamID.String()).Str("client_id", clientID.String()).Msg("Add client")
	}
	ch := make(chan av.Packet, 100)
	stream.Clients[clientID] = viewer{c: ch}
	return clientID, ch, nil
}

func (streams *StreamsStorage) deleteClient(streamID, clientID uuid.UUID) {
	streams.Lock()
	defer streams.Unlock()
	stream, ok := streams.Streams[streamID]
	if !ok {
		return
	}
	if stream.verboseLevel > VERBOSE_SIMPLE {
		log.Info().Str("scope", SCOPE_STREAM).Str("event", "client_delete").Str("stream_id", streamID.String()).Str("client_id", clientID.String()).Msg("Delete client")
	}
	delete(stream.Clients, clientID)
}

func (streams *StreamsStorage) cast(streamID uuid.UUID, pck av.Packet, hlsEnabled, archiveEnabled bool) error {
	streams.Lock()
	defer streams.Unlock()
	stream, ok := streams.Streams[streamID]
	if !ok {
		return ErrStreamNotFound
	}
	if stream.verboseLevel > VERBOSE_ADD {
		log.Info().Str("scope", SCOPE_STREAM).Str("event", "cast").Str("stream_id", streamID.String()).Bool("hls_enabled", hlsEnabled).Bool("archive_enabled", stream.archive != nil).Int("clients_num", len(stream.Clients)).Msg("Cast packet")
	}
	if hlsEnabled {
		stream.hlsChanel <- pck
	}
	if archiveEnabled {
		stream.mp4Chanel <- pck
	}
	for _, v := range stream.Clients {
		if len(v.c) < cap(v.c) {
			v.c <- pck
		}
	}
	return nil
}

func (streams *StreamsStorage) setArchiveStream(streamID uuid.UUID, streamArchiveConf configuration.StreamArchiveConfiguration) error {
	switch streamArchiveConf.TypeArchive {
	case "filesystem":
		err := streams.setArchiveStreamFS(streamID, streamArchiveConf.Directory, streamArchiveConf.MsPerSegment)
		if err != nil {
			return err
		}
	case "minio":
		err := streams.setArchiveStreamMinio(streamID, streamArchiveConf.Directory, streamArchiveConf.MinioBucket, streamArchiveConf.MsPerSegment)
		if err != nil {
			return err
		}
	default:

		return ErrNotSupportedStorage
	}

	return nil
}

func (streams *StreamsStorage) setArchiveStreamFS(streamID uuid.UUID, dir string, msPerSegment int64) error {
	if dir == "" {
		return fmt.Errorf("bad directory archive stream")
	}
	if msPerSegment == 0 {
		return fmt.Errorf("bad ms per segment archive stream")
	}
	newArhive := streamArhive{
		typeArchive:  "fylesystem",
		dir:          dir,
		msPerSegment: msPerSegment,
	}
	streams.Lock()
	defer streams.Unlock()
	stream, ok := streams.Streams[streamID]
	if !ok {
		return ErrStreamNotFound
	}
	stream.archive = &newArhive
	if stream.verboseLevel > VERBOSE_NONE {
		log.Info().Str("scope", SCOPE_STREAM).Str("event", "set_archive").Str("stream_id", streamID.String()).Str("dir", dir).Int64("ms_per_segment", msPerSegment).Msg("Add fylesystem archive")
	}
	return nil
}

func (streams *StreamsStorage) setArchiveStreamMinio(streamID uuid.UUID, dir string, bucket string, msPerSegment int64) error {
	if msPerSegment == 0 {
		return fmt.Errorf("bad ms per segment archive stream")
	}
	err := streams.minioStorage.MakeBucket(bucket)
	if err != nil {
		return ErrMakeBucket
	}

	newArhive := streamArhive{
		typeArchive:  "minio",
		dir:          dir,
		bucket:       bucket,
		msPerSegment: msPerSegment,
	}
	streams.Lock()
	defer streams.Unlock()
	stream, ok := streams.Streams[streamID]
	if !ok {
		return ErrStreamNotFound
	}
	stream.archive = &newArhive
	if stream.verboseLevel > VERBOSE_NONE {
		log.Info().Str("scope", SCOPE_STREAM).Str("event", "set_archive").Str("stream_id", streamID.String()).Str("bucket", bucket).Int64("ms_per_segment", msPerSegment).Msg("Add minio archive")
	}
	return nil
}

func (streams *StreamsStorage) getArchiveStream(streamID uuid.UUID) *streamArhive {
	streams.Lock()
	defer streams.Unlock()
	stream, ok := streams.Streams[streamID]
	if !ok {
		return nil
	}
	return stream.archive
}

func (streams *StreamsStorage) initMinio(minioSettings configuration.MinioSettings) error {
	minioStorage, err := storage.NewMinioProvider(
		fmt.Sprintf("%s:%d", minioSettings.Host, minioSettings.Port),
		minioSettings.User,
		minioSettings.Password,
		false,
		minioSettings.DefaultBucket,
		minioSettings.Path,
	)
	if err != nil {
		log.Error().Str("scope", "minio").Err(err).Msg("Can't init minio_storage")
		return err
	}
	err = minioStorage.Connect()
	if err != nil {
		log.Error().Str("scope", "minio").Err(err).Msg("Can't connect minio")
		return err
	}
	streams.minioStorage = minioStorage
	log.Info().Str("scope", "storage").Msg("Runing minio image storage")
	return nil
}

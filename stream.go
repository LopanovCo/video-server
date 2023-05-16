package videoserver

import (
	"bufio"
	"net/http"
	"time"

	"github.com/deepch/vdk/av"
	"github.com/deepch/vdk/format/rtspv2"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

const (
	pingDuration        = 15 * time.Second
	pingDurationRestart = pingDuration + 1*time.Second
	dialTimeoutDuration = 33 * time.Second
	readTimeoutDuration = 33 * time.Second
)

// runStream runs RTSP grabbing process
func (app *Application) runStream(streamID uuid.UUID, url string, hlsEnabled bool) error {
	session, err := rtspv2.Dial(rtspv2.RTSPClientOptions{
		URL:              url,
		DisableAudio:     true,
		DialTimeout:      dialTimeoutDuration,
		ReadWriteTimeout: readTimeoutDuration,
		Debug:            false,
	})
	if err != nil {
		return errors.Wrapf(err, "Can't connect to stream '%s'", url)
	}
	defer session.Close()
	if session.CodecData != nil {
		app.addCodec(streamID, session.CodecData)
		err = app.updateStreamStatus(streamID, true)
		if err != nil {
			return errors.Wrapf(err, "Can't update status for stream %s", streamID)
		}
	}
	isAudioOnly := false
	if len(session.CodecData) == 1 {
		if session.CodecData[0].Type().IsAudio() {
			isAudioOnly = true
		}
	}

	var stopHlsCast chan bool
	var stopMP4Cast chan bool
	if hlsEnabled {
		stopHlsCast = make(chan bool, 1)
		app.startHlsCast(streamID, stopHlsCast)
	}
	mp4Enable := true
	if mp4Enable {
		stopMP4Cast = make(chan bool, 1)
		app.startMP4Cast(streamID, stopMP4Cast)
	}
	pingStream := time.NewTimer(pingDuration)
	for {
		select {
		case <-pingStream.C:
			return errors.Wrapf(ErrStreamHasNoVideo, "URL is '%s'", url)
		case signals := <-session.Signals:
			switch signals {
			case rtspv2.SignalCodecUpdate:
				app.addCodec(streamID, session.CodecData)
				err = app.updateStreamStatus(streamID, true)
				if err != nil {
					return errors.Wrapf(err, "Can't update status for stream %s", streamID)
				}
			case rtspv2.SignalStreamRTPStop:
				err = app.updateStreamStatus(streamID, false)
				if err != nil {
					errors.Wrapf(err, "Can't switch status to False for stream '%s'", url)
				}
				return errors.Wrapf(ErrStreamDisconnected, "URL is '%s'", url)
			}
		case packetAV := <-session.OutgoingPacketQueue:
			if isAudioOnly || packetAV.IsKeyFrame {
				pingStream.Reset(pingDurationRestart)
			}
			err = app.cast(streamID, *packetAV, hlsEnabled)
			if err != nil {
				if hlsEnabled {
					stopHlsCast <- true
				}
				if mp4Enable {
					stopMP4Cast <- true
				}
				errStatus := app.updateStreamStatus(streamID, false)
				if errStatus != nil {
					errors.Wrapf(errors.Wrapf(err, "Can't cast packet %s (%s)", streamID, url), "Can't switch status to False for stream '%s'", url)
				}
				return errors.Wrapf(err, "Can't cast packet %s (%s)", streamID, url)
			}
		}
	}
}

func (app *Application) runMJPEGStream(streamID uuid.UUID, url string, hlsEnabled bool) error {
	timeStart := time.Now()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	reader := bufio.NewReader(resp.Body)
	var idx int8 = 0
	for {
		timeDecode := time.Now()
		data := make([]byte, 100000)
		n, err := reader.Read(data)
		if err != nil {
			return errors.Wrapf(err, "Can't read body data for stream id %s (%s)", streamID, url)
		}
		decode := time.Since(timeDecode)
		newPcketAV := av.Packet{
			IsKeyFrame:      true,
			Idx:             idx,
			CompositionTime: time.Since(timeStart) - decode,
			Time:            decode,
			Duration:        time.Since(timeStart),
			Data:            data[:n],
		}
		err = app.cast(streamID, newPcketAV, hlsEnabled)
		if err != nil {
			errStatus := app.updateStreamStatus(streamID, false)
			if errStatus != nil {
				errors.Wrapf(errors.Wrapf(err, "Can't cast packet %s (%s)", streamID, url), "Can't switch status to False for stream '%s'", url)
			}
			return errors.Wrapf(err, "Can't cast packet %s (%s)", streamID, url)
		}
		idx++
	}
	return nil
}

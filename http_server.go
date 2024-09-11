package videoserver

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// StartAPIServer starts server with API functionality
func (app *Application) StartAPIServer() {
	router := gin.New()

	pprof.Register(router)

	if app.CorsConfig != nil {
		router.Use(cors.New(*app.CorsConfig))
	}
	router.GET("/list", ListWrapper(app, app.APICfg.Verbose))
	router.GET("/status", StatusWrapper(app, app.APICfg.Verbose))
	router.POST("/enable_camera", EnableCamera(app, app.APICfg.Verbose))
	router.POST("/disable_camera", DisableCamera(app, app.APICfg.Verbose))

	url := fmt.Sprintf("%s:%d", app.APICfg.Host, app.APICfg.Port)
	s := &http.Server{
		Addr:         url,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	if app.APICfg.Verbose > VERBOSE_NONE {
		log.Info().Str("scope", "api_server").Str("event", "api_server_start").Str("url", url).Msg("Start microservice for API server")
	}
	err := s.ListenAndServe()
	if err != nil {
		if app.APICfg.Verbose > VERBOSE_NONE {
			log.Error().Err(err).Str("scope", "api_server").Str("event", "api_server_start").Str("url", url).Msg("Can't start API server routers")
		}
		return
	}
}

type StreamsInfoShortenList struct {
	Data []StreamInfoShorten `json:"data"`
}

type StreamInfoShorten struct {
	StreamID string `json:"stream_id"`
}

// ListWrapper returns list of streams
func ListWrapper(app *Application, verboseLevel VerboseLevel) func(ctx *gin.Context) {
	return func(ctx *gin.Context) {
		if verboseLevel > VERBOSE_SIMPLE {
			log.Info().Str("scope", "api_server").Str("method", ctx.Request.Method).Str("route", ctx.Request.URL.Path).Str("remote", ctx.Request.RemoteAddr).Msg("Call streams list")
		}
		allStreamsIDs := app.getStreamsIDs()
		ans := StreamsInfoShortenList{
			Data: make([]StreamInfoShorten, len(allStreamsIDs)),
		}
		for i, streamID := range allStreamsIDs {
			ans.Data[i] = StreamInfoShorten{
				StreamID: streamID.String(),
			}
		}
		ctx.JSON(200, ans)
	}
}

// StatusWrapper returns statuses for list of streams
func StatusWrapper(app *Application, verboseLevel VerboseLevel) func(ctx *gin.Context) {
	return func(ctx *gin.Context) {
		if verboseLevel > VERBOSE_SIMPLE {
			log.Info().Str("scope", "api_server").Str("method", ctx.Request.Method).Str("route", ctx.Request.URL.Path).Str("remote", ctx.Request.RemoteAddr).Msg("Call streams' statuses list")
		}
		ctx.JSON(200, app)
	}
}

// EnablePostData is a POST-body for API which enables to turn on/off specific streams
type EnablePostData struct {
	GUID        uuid.UUID `json:"guid"`
	URL         string    `json:"url"`
	OutputTypes []string  `json:"output_types"`
}

// EnableCamera adds new stream if does not exist
func EnableCamera(app *Application, verboseLevel VerboseLevel) func(ctx *gin.Context) {
	return func(ctx *gin.Context) {
		if verboseLevel > VERBOSE_SIMPLE {
			log.Info().Str("scope", "api_server").Str("method", ctx.Request.Method).Str("route", ctx.Request.URL.Path).Str("remote", ctx.Request.RemoteAddr).Msg("Try to enable camera")
		}
		var postData EnablePostData
		if err := ctx.ShouldBindJSON(&postData); err != nil {
			errReason := "Bad JSON binding"
			if verboseLevel > VERBOSE_NONE {
				log.Error().Err(err).Str("scope", "api_server").Str("method", ctx.Request.Method).Str("route", ctx.Request.URL.Path).Str("remote", ctx.Request.RemoteAddr).Msg(errReason)
			}
			ctx.JSON(http.StatusBadRequest, gin.H{"Error": errReason})
			return
		}
		if exist := app.streamExists(postData.GUID); !exist {
			outputTypes := make([]StreamType, 0, len(postData.OutputTypes))
			for _, v := range postData.OutputTypes {
				typ, ok := streamTypeExists(v)
				if !ok {
					errReason := fmt.Sprintf("%s. Type: '%s'", ErrStreamTypeNotExists, v)
					if verboseLevel > VERBOSE_NONE {
						log.Error().Err(fmt.Errorf(errReason)).Str("scope", "api_server").Str("method", ctx.Request.Method).Str("route", ctx.Request.URL.Path).Str("remote", ctx.Request.RemoteAddr).Msg(errReason)
					}
					ctx.JSON(http.StatusBadRequest, gin.H{"Error": errReason})
					return
				}
				if _, ok := supportedOutputStreamTypes[typ]; !ok {
					errReason := fmt.Sprintf("%s. Type: '%s'", ErrStreamTypeNotSupported, v)
					if verboseLevel > VERBOSE_NONE {
						log.Error().Err(fmt.Errorf(errReason)).Str("scope", "api_server").Str("method", ctx.Request.Method).Str("route", ctx.Request.URL.Path).Str("remote", ctx.Request.RemoteAddr).Msg(errReason)
					}
					ctx.JSON(http.StatusBadRequest, gin.H{"Error": errReason})
					return
				}
				outputTypes = append(outputTypes, typ)
			}
			app.Streams.Lock()
			app.Streams.Streams[postData.GUID] = NewStreamConfiguration(postData.URL, outputTypes)
			app.Streams.Unlock()
			app.StartStream(postData.GUID)
		}
		ctx.JSON(200, app)
	}
}

// DisableCamera turns off stream for specific stream ID
func DisableCamera(app *Application, verboseLevel VerboseLevel) func(ctx *gin.Context) {
	return func(ctx *gin.Context) {
		if verboseLevel > VERBOSE_SIMPLE {
			log.Info().Str("scope", "api_server").Str("method", ctx.Request.Method).Str("route", ctx.Request.URL.Path).Str("remote", ctx.Request.RemoteAddr).Msg("Try to disable camera")
		}
		var postData EnablePostData
		if err := ctx.ShouldBindJSON(&postData); err != nil {
			errReason := "Bad JSON binding"
			if verboseLevel > VERBOSE_NONE {
				log.Error().Err(err).Str("scope", "api_server").Str("method", ctx.Request.Method).Str("route", ctx.Request.URL.Path).Str("remote", ctx.Request.RemoteAddr).Msg(errReason)
			}
			ctx.JSON(http.StatusBadRequest, gin.H{"Error": errReason})
			return
		}
		if exist := app.streamExists(postData.GUID); exist {
			app.Streams.Lock()
			delete(app.Streams.Streams, postData.GUID)
			app.Streams.Unlock()
		}
		ctx.JSON(200, app)
	}
}

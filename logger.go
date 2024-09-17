package videoserver

const (
	SCOPE_APP        = "app"
	SCOPE_STREAM     = "stream"
	SCOPE_WS_HANDLER = "ws_handler"
	SCOPE_API_SERVER = "api_server"
	SCOPE_WS_SERVER  = "ws_server"
	SCOPE_ARCHIVE    = "archive"
	SCOPE_MP4        = "mp4"

	EVENT_APP_CORS_CONFIG = "app_cors_config"

	EVENT_API_PREPARE     = "api_server_prepare"
	EVENT_API_START       = "api_server_start"
	EVENT_API_CORS_ENABLE = "api_server_cors_enable"
	EVENT_API_REQUEST     = "api_request"

	EVENT_WS_PREPARE     = "ws_server_prepare"
	EVENT_WS_START       = "ws_server_start"
	EVENT_WS_CORS_ENABLE = "ws_server_cors_enable"
	EVENT_WS_REQUEST     = "ws_request"
	EVENT_WS_UPGRADER    = "ws_upgrader"

	EVENT_ARCHIVE_CREATE_FILE = "archive_create_file"
	EVENT_ARCHIVE_CLOSE_FILE  = "archive_close_file"
	EVENT_MP4_WRITE           = "mp4_write"
	EVENT_MP4_WRITE_TRAIL     = "mp4_write_trail"
	EVENT_MP4_SAVE_MINIO      = "mp4_save_minio"
	EVENT_MP4_CLOSE           = "mp4_close"
)

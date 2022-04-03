package main

import (
	"log"
	"os"
	"reflect"
	"strconv"

	"github.com/hammertrack/tracker/errors"
	"github.com/joho/godotenv"
)

var ErrUnsupportedSecretLength = errors.New("secret length must be 256 bits or 32 bytes")

var (
	APIPort              string
	DBHost               string
	DBKeyspace           string
	DBPort               string
	DBUser               string
	DBPassword           string
	DBConnTimeoutSeconds int
	DBPageSize           int
	DBCursorSecretString string
	DBCursorSecret       []byte

	Debug                 bool
	ServerReadTimeout     int
	ServerWriteTimeout    int
	ServerIdleTimeout     int
	ServerReadBufferSize  int
	ServerWriteBufferSize int
	ServerProxyHeader     string
	ServerBodyLimitBytes  int
	ServerConcurrency     int
)

type SupportStringconv interface {
	~int | ~int64 | ~float32 | ~string | ~bool
}

func conv(v string, to reflect.Kind) any {
	var err error

	if to == reflect.String {
		return v
	}

	if to == reflect.Bool {
		if bool, err := strconv.ParseBool(v); err == nil {
			return bool
		}
	}

	if to == reflect.Int {
		if int, err := strconv.Atoi(v); err == nil {
			return int
		}
	}

	if to == reflect.Int64 {
		if i64, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i64
		}
	}

	if to == reflect.Float32 {
		if f32, err := strconv.ParseFloat(v, 32); err == nil {
			return f32
		}
	}

	errors.WrapFatalWithContext(err, struct {
		EnvKey string
	}{v})
	return nil
}

func Env[T SupportStringconv](key string, def T) T {
	if v, ok := os.LookupEnv(key); ok {
		val := conv(v, reflect.TypeOf(def).Kind()).(T)
		log.Printf("[%s] => %v", key, val)
		return val
	}
	return def
}

func LoadConfig() {
	if err := godotenv.Load(); err != nil {
		errors.WrapFatal(err)
	}

	APIPort = Env("API_PORT", "3000")
	DBHost = Env("DB_HOST", "127.0.0.1")
	DBKeyspace = Env("DB_KEYSPACE", "hammertrack")
	DBPort = Env("DB_PORT", "5200")
	DBUser = Env("DB_USER", "tracker")
	DBPassword = Env("DB_PASSWORD", "unsafepassword")
	DBConnTimeoutSeconds = Env("DB_CONN_TIMEOUT_SECONDS", 20)
	DBPageSize = Env("DB_PAGE_SIZE", 200)
	DBCursorSecretString = Env("DB_CURSOR_SECRET", "unsafesecret")
	DBCursorSecret = []byte(DBCursorSecretString)

	Debug = Env("DEBUG", false)
	ServerReadTimeout = Env("SERVER_READ_TIMEOUT", 1)
	ServerWriteTimeout = Env("SERVER_WRITE_TIMEOUT", 1)
	ServerIdleTimeout = Env("SERVER_IDLE_TIMEOUT", 30)
	ServerReadBufferSize = Env("SERVER_READ_BUFFER_SIZE", 4096)
	ServerWriteBufferSize = Env("SERVER_WRITE_BUFFER_SIZE", 4096)
	ServerProxyHeader = Env("SERVER_PROXY_HEADER", "")
	ServerBodyLimitBytes = Env("SERVER_BODY_LIMIT_BYTES", 1*1024*1024)
	ServerConcurrency = Env("SERVER_CONCURRENCY", 256*1024)

	if len(DBCursorSecret) != 32 {
		errors.WrapFatal(ErrUnsupportedSecretLength)
	}
}

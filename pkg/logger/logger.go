package logger

import (
	"errors"
	"os"

	"github.com/datreeio/admission-webhook-datree/pkg/errorReporter"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	admission "k8s.io/api/admission/v1"
)

// most of our logs are in the following places:
// 1. webhook start up
// 2. incoming request
// 3. outgoing request
// 4. errors

// Logger - instructions to get the logs are under /guides/developer-guide.md
type Logger struct {
	zapLogger     *zap.Logger
	requestId     string
	errorReporter *errorReporter.ErrorReporter
}

func New(requestId string, errorReporter *errorReporter.ErrorReporter) Logger {
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	jsonEncoder := zapcore.NewJSONEncoder(config)

	defaultLogLevel := zapcore.DebugLevel

	core := zapcore.NewTee(zapcore.NewCore(jsonEncoder, zapcore.AddSync(os.Stdout), defaultLogLevel))

	zapLogger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return Logger{
		zapLogger:     zapLogger,
		requestId:     requestId,
		errorReporter: errorReporter,
	}
}

func (l *Logger) LogError(message string) {
	l.zapLogger.Error(message, zap.String("requestId", l.requestId))
}

func (l *Logger) LogAndReportUnexpectedError(message string) {
	l.LogError(message)
	l.errorReporter.ReportUnexpectedError(errors.New(message))
}

func (l *Logger) LogIncoming(admissionReview *admission.AdmissionReview) {
	l.logInfo(admissionReview, "incoming")
}
func (l *Logger) LogOutgoing(admissionReview *admission.AdmissionReview, isSkipped bool) {
	l.logInfo(outgoingLog{
		AdmissionReview: admissionReview,
		IsSkipped:       isSkipped,
	}, "outgoing")
}

type outgoingLog struct {
	AdmissionReview *admission.AdmissionReview
	IsSkipped       bool
}

func (l *Logger) LogInfo(objectToLog any) {
	l.logInfo(objectToLog, "")
}

// LogUtil this method creates a new logger instance on every call, and does not have a requestId
// please prefer using the logger instance from the context instead
func LogUtil(msg string) {
	logger := New("", nil)
	logger.LogInfo(msg)
}

func (l *Logger) logInfo(objectToLog any, requestDirection string) {
	logFields := make(map[string]interface{})
	logFields["requestId"] = l.requestId
	logFields["requestDirection"] = requestDirection
	logFields["msg"] = objectToLog

	l.zapLogger.Info("Logging information", zap.Any("logFields", logFields))
}

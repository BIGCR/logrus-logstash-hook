package logrustash

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"io"
)

type simpleFmter struct{}

func (f simpleFmter) Format(e *logrus.Entry) ([]byte, error) {
	return []byte(fmt.Sprintf("msg: %#v", e.Message)), nil
}

func TestFire(t *testing.T) {
	buffer := bytes.NewBuffer(nil)
	h := Hook{
		writer:    buffer,
		formatter: simpleFmter{},
	}

	entry := &logrus.Entry{
		Message: "my message",
		Data:    logrus.Fields{},
	}

	err := h.Fire(entry)
	if err != nil {
		t.Error("expected Fire to not return error")
	}

	expected := "msg: \"my message\""
	if buffer.String() != expected {
		t.Errorf("expected to see '%s' in '%s'", expected, buffer.String())
	}
}

type FailFmt struct{}

func (f FailFmt) Format(e *logrus.Entry) ([]byte, error) {
	return nil, errors.New("")
}

func TestFireFormatError(t *testing.T) {
	buffer := bytes.NewBuffer(nil)
	h := Hook{
		writer:    buffer,
		formatter: FailFmt{},
	}

	if err := h.Fire(&logrus.Entry{Data: logrus.Fields{}}); err == nil {
		t.Error("expected Fire to return error")
	}
}

type FailWrite struct{}

func (w FailWrite) Write(d []byte) (int, error) {
	return 0, errors.New("")
}

func TestFireWriteError(t *testing.T) {
	h := Hook{
		writer:    FailWrite{},
		formatter: &logrus.JSONFormatter{},
	}

	if err := h.Fire(&logrus.Entry{Data: logrus.Fields{}}); err == nil {
		t.Error("expected Fire to return error")
	}
}

func TestDefaultFormatterWithFields(t *testing.T) {
	format := DefaultFormatter(logrus.Fields{"ID": 123})

	entry := &logrus.Entry{
		Message: "msg1",
		Data:    logrus.Fields{"f1": "bla"},
	}

	res, err := format.Format(entry)
	if err != nil {
		t.Errorf("expected format to not return error: %s", err)
	}

	expected := []string{
		"f1\":\"bla\"",
		"ID\":123",
		"message\":\"msg1\"",
	}

	for _, exp := range expected {
		if !strings.Contains(string(res), exp) {
			t.Errorf("expected to have '%s' in '%s'", exp, string(res))
		}
	}
}

func TestDefaultFormatterWithEmptyFields(t *testing.T) {
	now := time.Now()
	formatter := DefaultFormatter(logrus.Fields{})

	entry := &logrus.Entry{
		Message: "message bla bla",
		Level:   logrus.DebugLevel,
		Time:    now,
		Data: logrus.Fields{
			"Key1": "Value1",
		},
	}

	res, err := formatter.Format(entry)
	if err != nil {
		t.Errorf("expected Format not to return error: %s", err)
	}

	expected := []string{
		"\"message\":\"message bla bla\"",
		"\"level\":\"debug\"",
		"\"Key1\":\"Value1\"",
		"\"@version\":\"1\"",
		"\"type\":\"log\"",
		fmt.Sprintf("\"@timestamp\":\"%s\"", now.Format(logrus.DefaultTimestampFormat)),
	}

	for _, exp := range expected {
		if !strings.Contains(string(res), exp) {
			t.Errorf("expected to have '%s' in '%s'", exp, string(res))
		}
	}
}

func TestLogstashFieldsNotOverridden(t *testing.T) {
	_ = DefaultFormatter(logrus.Fields{"user1": "11"})

	if _, ok := logstashFields["user1"]; ok {
		t.Errorf("expected user1 to not be in logstashFields: %#v", logstashFields)
	}
}

func TestFireWithLevels(t *testing.T) {
	buffer := bytes.NewBuffer(nil)
	h := Hook{
		writer:    buffer,
		formatter: simpleFmter{},
	}

	h.SetLevel(logrus.WarnLevel)

	testData := []struct {
		writer   io.Writer
		level    logrus.Level
		message  string
		expected string
	}{
		{
			bytes.NewBuffer(nil),
			logrus.DebugLevel,
			"debug",
			"",
		},
		{
			bytes.NewBuffer(nil),
			logrus.WarnLevel,
			"warn",
			"msg: \"warn\"",
		},
	}

	for _, test := range testData {
		entry := &logrus.Entry{
			Message: test.message,
			Data:    logrus.Fields{},
			Level:   test.level,
		}

		err := h.Fire(entry)
		if err != nil {
			t.Error("expected Fire to not return error")
		}

		if buffer.String() != test.expected {
			t.Errorf("expected to see '%s' in '%s'", test.expected, buffer.String())
		}
	}
}

func TestHook_RemoveLevel(t *testing.T) {
	hook := Hook{
		levels: logrus.AllLevels,
	}

	for _, levelToRemove := range logrus.AllLevels {
		hook.RemoveLevel(levelToRemove)

		for _, level := range hook.levels {
			if level == levelToRemove {
				t.Errorf("Level %d was not removed from hook levels %v", levelToRemove, hook.levels)
			}
		}
	}
}

func TestHook_SetLevel(t *testing.T) {
	hook := Hook{
		levels: []logrus.Level{},
	}

	for _, levelToAdd := range logrus.AllLevels {
		hook.SetLevel(levelToAdd)

		var found bool = false

		for _, level := range hook.levels {
			if level == levelToAdd {
				found = true
			}
		}

		if !found {
			t.Errorf("Level %d was not added to hook levels %v", levelToAdd, hook.levels)
		}
	}
}

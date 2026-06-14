package nats_test

import (
	"testing"

	"github.com/sahiy/sahiy-stream/pkg/nats"
	"github.com/sahiy/sahiy-stream/pkg/transcode"
)

func TestTranscodeStreamNames(t *testing.T) {
	if nats.TranscodeCmdStream != "TRANSCODE_CMD" {
		t.Fatalf("cmd stream: %s", nats.TranscodeCmdStream)
	}
	if nats.TranscodeEvtStream != "TRANSCODE_EVT" {
		t.Fatalf("evt stream: %s", nats.TranscodeEvtStream)
	}
	if nats.TranscodeWorkersQ != "transcode-workers" {
		t.Fatalf("queue: %s", nats.TranscodeWorkersQ)
	}
}

func TestTranscodeSubjects(t *testing.T) {
	cases := map[string]string{
		transcode.CmdStartSubject: "transcode.cmd.start",
		transcode.CmdStopSubject:  "transcode.cmd.stop",
	}
	for got, want := range cases {
		if got != want {
			t.Fatalf("subject %q want %q", got, want)
		}
	}
	if transcode.EvtSubject("stream-1") != "transcode.evt.stream-1" {
		t.Fatal("evt subject mismatch")
	}
}

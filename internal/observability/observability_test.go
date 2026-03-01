// Copyright 2026 Pidgr, Inc. All rights reserved.
// Licensed under the Apache License, Version 2.0.

package observability

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
)

func TestInitTracer_NoEndpoint_ReturnsNoOpProvider(t *testing.T) {
	tp, err := InitTracer(context.Background(), "", "pidgr-mcp")
	if err != nil {
		t.Fatalf("InitTracer returned error: %v", err)
	}
	if tp == nil {
		t.Fatal("expected non-nil TracerProvider")
	}
	if err := tp.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown error: %v", err)
	}
}

func TestInitTracer_WithEndpoint_ReturnsProvider(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")

	tp, err := InitTracer(context.Background(), "http://localhost:4318", "pidgr-mcp")
	if err != nil {
		t.Fatalf("InitTracer returned error: %v", err)
	}
	if tp == nil {
		t.Fatal("expected non-nil TracerProvider")
	}
	if err := tp.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown error: %v", err)
	}
}

func TestInitLogger_NoEndpoint_ReturnsNoOpProvider(t *testing.T) {
	lp, err := InitLogger(context.Background(), "", "pidgr-mcp")
	if err != nil {
		t.Fatalf("InitLogger returned error: %v", err)
	}
	if lp == nil {
		t.Fatal("expected non-nil LoggerProvider")
	}
	if err := lp.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown error: %v", err)
	}
}

func TestInitLogger_WithEndpoint_ReturnsProvider(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318")

	lp, err := InitLogger(context.Background(), "http://localhost:4318", "pidgr-mcp")
	if err != nil {
		t.Fatalf("InitLogger returned error: %v", err)
	}
	if lp == nil {
		t.Fatal("expected non-nil LoggerProvider")
	}
	if err := lp.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown error: %v", err)
	}
}

func TestFanoutHandler_Enabled(t *testing.T) {
	infoHandler := slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelInfo})
	warnHandler := slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelWarn})
	fanout := NewFanoutHandler(infoHandler, warnHandler)

	if !fanout.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("expected Enabled=true for Info (infoHandler accepts it)")
	}
	if fanout.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("expected Enabled=false for Debug (neither handler accepts it)")
	}
	if !fanout.Enabled(context.Background(), slog.LevelWarn) {
		t.Error("expected Enabled=true for Warn (both handlers accept it)")
	}
}

func TestFanoutHandler_Handle(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	h1 := slog.NewJSONHandler(&buf1, nil)
	h2 := slog.NewJSONHandler(&buf2, nil)
	fanout := NewFanoutHandler(h1, h2)

	logger := slog.New(fanout)
	logger.Info("test message", "key", "value")

	if !bytes.Contains(buf1.Bytes(), []byte("test message")) {
		t.Error("handler 1 did not receive the message")
	}
	if !bytes.Contains(buf2.Bytes(), []byte("test message")) {
		t.Error("handler 2 did not receive the message")
	}
}

func TestFanoutHandler_LevelFiltering(t *testing.T) {
	var infoBuf, warnBuf bytes.Buffer
	infoHandler := slog.NewJSONHandler(&infoBuf, &slog.HandlerOptions{Level: slog.LevelInfo})
	warnHandler := slog.NewJSONHandler(&warnBuf, &slog.HandlerOptions{Level: slog.LevelWarn})
	fanout := NewFanoutHandler(infoHandler, warnHandler)

	logger := slog.New(fanout)
	logger.Info("info only")

	if !bytes.Contains(infoBuf.Bytes(), []byte("info only")) {
		t.Error("info handler should have received the Info message")
	}
	if warnBuf.Len() != 0 {
		t.Error("warn handler should not have received an Info message")
	}
}

func TestFanoutHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, nil)
	fanout := NewFanoutHandler(h)

	withAttrs := fanout.WithAttrs([]slog.Attr{slog.String("service", "test")})
	logger := slog.New(withAttrs)
	logger.Info("attr test")

	if !bytes.Contains(buf.Bytes(), []byte("service")) {
		t.Error("expected 'service' attr in output")
	}
}

func TestFanoutHandler_WithGroup(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, nil)
	fanout := NewFanoutHandler(h)

	withGroup := fanout.WithGroup("mygroup")
	logger := slog.New(withGroup)
	logger.Info("group test", "key", "val")

	if !bytes.Contains(buf.Bytes(), []byte("mygroup")) {
		t.Error("expected 'mygroup' in output")
	}
}

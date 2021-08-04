// Copyright 2020 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package workflow_test

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"golang.org/x/build/internal/workflow"
)

func TestTrivial(t *testing.T) {
	wd := workflow.New()
	greeting := wd.Task("echo", echo, wd.Constant("hello world"))
	wd.Output("greeting", greeting)

	w, err := workflow.Start(wd, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := w.Run(context.Background(), loggingListener(t))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := outputs["greeting"], "hello world"; got != want {
		t.Errorf("greeting = %q, want %q", got, want)
	}
}

func TestSplitJoin(t *testing.T) {
	wd := workflow.New()
	in := wd.Task("echo", echo, wd.Constant("string #"))
	add1 := wd.Task("add 1", appendInt, in, wd.Constant(1))
	add2 := wd.Task("add 2", appendInt, in, wd.Constant(2))
	both := wd.Slice([]workflow.Value{add1, add2})
	out := wd.Task("join", join, both)
	wd.Output("strings", out)

	w, err := workflow.Start(wd, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := w.Run(context.Background(), loggingListener(t))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := outputs["strings"], "string #1,string #2"; got != want {
		t.Errorf("joined output = %q, want %q", got, want)
	}
}

func TestParallelism(t *testing.T) {
	wd := workflow.New()
	out1 := wd.Task("sleep #1", sleep, wd.Constant(100*time.Millisecond))
	out2 := wd.Task("sleep #2", sleep, wd.Constant(100*time.Millisecond))
	wd.Output("out1", out1)
	wd.Output("out2", out2)

	w, err := workflow.Start(wd, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	_, err = w.Run(context.Background(), loggingListener(t))
	if err != nil {
		t.Fatal(err)
	}
	if delay := time.Since(start); delay > 150*time.Millisecond {
		t.Errorf("too much time elapsed: %v", delay)
	}
}

func TestParameters(t *testing.T) {
	wd := workflow.New()
	param1 := wd.Parameter("param1")
	param2 := wd.Parameter("param2")
	out1 := wd.Task("echo 1", echo, param1)
	out2 := wd.Task("echo 2", echo, param2)
	wd.Output("out1", out1)
	wd.Output("out2", out2)

	w, err := workflow.Start(wd, map[string]string{"param1": "#1", "param2": "#2"})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := w.Run(context.Background(), loggingListener(t))
	if err != nil {
		t.Fatal(err)
	}
	if want := map[string]interface{}{"out1": "#1", "out2": "#2"}; !reflect.DeepEqual(outputs, want) {
		t.Errorf("outputs = %#v, want %#v", outputs, want)
	}
}

func sleep(ctx context.Context, d time.Duration) (struct{}, error) {
	time.Sleep(d)
	return struct{}{}, nil
}

func appendInt(ctx context.Context, s string, i int) (string, error) {
	return fmt.Sprintf("%v%v", s, i), nil
}

func join(ctx context.Context, s []string) (string, error) {
	return strings.Join(s, ","), nil
}

func echo(ctx context.Context, arg string) (string, error) {
	return arg, nil
}

func loggingListener(t *testing.T) func(*workflow.TaskState) {
	return func(st *workflow.TaskState) {
		switch {
		case !st.Finished:
			t.Logf("task %-10v: started", st.Name)
		case st.Error != nil:
			t.Logf("task %-10v: error: %v", st.Name, st.Error)
		default:
			t.Logf("task %-10v: done: %v", st.Name, st.Result)
		}
	}
}

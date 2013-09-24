package log
// Copyright 2013, David Fisher. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// author: David Fisher <ddf1991@gmail.com>
// based on previous package by: Cong Ding <dinggnu@gmail.com>

import (
	"fmt"
	"github.com/coreos/go-systemd/journal"
	"io"
	"strings"
	"sync"
)

const AsyncBuffer = 100

type Sink interface {
	Log(Fields)
}

type writerSink struct {
	lock   sync.Mutex
	out    io.Writer
	format string
	fields []string
}

func (sink *writerSink) Log(fields Fields) {
	vals := make([]interface{}, len(sink.fields))
	for i, field := range sink.fields {
		var ok bool
		vals[i], ok = fields[field]
		if !ok {
			vals[i] = "???"
		}
	}

	sink.lock.Lock()
	defer sink.lock.Unlock()
	fmt.Fprintf(sink.out, sink.format, vals...)
}

func WriterSink(out io.Writer, format string, fields []string) Sink {
	return &writerSink{
		out:    out,
		format: format,
		fields: fields,
	}
}

type journalSink struct{}

func (sink *journalSink) Log(fields Fields) {
	message := fields["message"].(string)
	priority := toJournalPriority(fields["priority"].(Priority))
	journalFields := make(map[string]string)
	for k, v := range fields {
		if k == "message" || k == "priority" {
			continue
		}
		journalFields[strings.ToUpper(k)] = fmt.Sprint(v)
	}
	journal.Send(message, priority, journalFields)
}

func toJournalPriority(priority Priority) journal.Priority {
	switch priority {
	case PriEmerg:
		return journal.PriEmerg
	case PriAlert:
		return journal.PriAlert
	case PriCrit:
		return journal.PriCrit
	case PriErr:
		return journal.PriErr
	case PriWarning:
		return journal.PriWarning
	case PriNotice:
		return journal.PriNotice
	case PriInfo:
		return journal.PriInfo
	case PriDebug:
		return journal.PriDebug

	default:
		return journal.PriErr
	}
}

func JournalSink() Sink {
	return &journalSink{}
}

type tryJournalSink struct {
	j journalSink
	w writerSink
}

func (sink *tryJournalSink) Log(fields Fields) {
	if journal.Enabled() {
		sink.j.Log(fields)
	} else {
		sink.w.Log(fields)
	}
}

func JournalFallbackSink(out io.Writer, format string, fields []string) Sink {
	return &tryJournalSink{
		w: writerSink{
			out:    out,
			format: format,
			fields: fields,
		},
	}
}
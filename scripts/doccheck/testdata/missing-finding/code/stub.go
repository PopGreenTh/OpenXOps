package code

import "github.com/PopGreenTh/OpenXOps/record"

func Stub() {
	addErr := func(code, subject, msg string) {
		_ = record.Finding("error", code, subject, msg, "cmd")
	}
	addErr("fixture-helper-code", "subj", "msg")
}

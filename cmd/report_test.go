package cmd

import (
	"strings"
	"testing"
)

func TestFormatChatText_InlineLatexUnits(t *testing.T) {
	input := "在初始化阶段，仅针对 $1\\text{GB}$ 与 $2\\text{MB}$ 级元数据表实施预分配，而将 `PAMT_PAGE_BITMAP` 位图保留。"
	got := formatChatText(input)
	if strings.Contains(got, "$") || strings.Contains(got, `\text`) {
		t.Fatalf("got = %q", got)
	}
	if !strings.Contains(got, "1GB") || !strings.Contains(got, "2MB") || !strings.Contains(got, "`PAMT_PAGE_BITMAP`") {
		t.Fatalf("got = %q", got)
	}
}

func TestFormatChatText_BlockMath(t *testing.T) {
	got := formatChatText("$$x$$")
	if got != "x" {
		t.Fatalf("got = %q", got)
	}
}

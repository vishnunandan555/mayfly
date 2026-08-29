package screen

import (
	"bytes"
	"strings"
	"testing"
)

func TestStyleSequenceSupportsOnlyNeededAttributesAndColors(t *testing.T) {
	style := Style{
		Foreground: ColorRed,
		Background: ColorBlue,
		Attributes: AttrBold | AttrDim | AttrUnderline | AttrReverse,
	}
	if got, want := style.Sequence(StyleConfig{ColorMode: ColorModeANSI}), "\x1b[1;2;4;7;31;44m"; got != want {
		t.Fatalf("basic style sequence = %q, want %q", got, want)
	}
	bright := Style{Foreground: ColorBrightRed, Background: ColorBrightWhite}
	if got, want := bright.Sequence(StyleConfig{ColorMode: ColorModeBright}), "\x1b[91;107m"; got != want {
		t.Fatalf("bright style sequence = %q, want %q", got, want)
	}
	if got, want := bright.Sequence(StyleConfig{ColorMode: ColorModeANSI}), "\x1b[31;47m"; got != want {
		t.Fatalf("basic fallback sequence = %q, want %q", got, want)
	}
}

func TestStyleResetAndNestedWritesDoNotLeakAttributes(t *testing.T) {
	var output bytes.Buffer
	terminal := NewTerminalWithConfig(&output, Size{Rows: 1, Columns: 20}, StyleConfig{ColorMode: ColorModeANSI})
	if err := terminal.WriteStyled(Style{Attributes: AttrBold}, "outer"); err != nil {
		t.Fatal(err)
	}
	if err := terminal.WriteStyled(Style{Foreground: ColorRed}, "inner"); err != nil {
		t.Fatal(err)
	}
	if err := terminal.Flush(); err != nil {
		t.Fatal(err)
	}
	want := "\x1b[1mouter\x1b[0m\x1b[31minner\x1b[0m"
	if got := output.String(); got != want {
		t.Fatalf("styled writes = %q, want %q", got, want)
	}

	if got := (Style{}).Sequence(StyleConfig{ColorMode: ColorModeANSI}); got != "" {
		t.Fatalf("zero style sequence = %q, want empty", got)
	}
	if got := (Style{Foreground: ColorRed}).Sequence(StyleConfig{ColorMode: ColorModeNone}); got != "" {
		t.Fatalf("color-only plain sequence = %q, want empty", got)
	}
	if got := (Style{Attributes: AttrUnderline}).Sequence(StyleConfig{ColorMode: ColorModeNone}); got != "\x1b[4m" {
		t.Fatalf("attribute-only plain sequence = %q, want underline", got)
	}
}

func TestStyleConfigurationDisablesAllStyling(t *testing.T) {
	style := Style{Foreground: ColorGreen, Attributes: AttrBold | AttrUnderline}
	if got := style.Sequence(StyleConfig{ColorMode: ColorModeBright, DisableStyling: true}); got != "" {
		t.Fatalf("disabled style sequence = %q, want empty", got)
	}

	var output bytes.Buffer
	terminal := NewTerminalWithConfig(&output, Size{Rows: 1, Columns: 8}, StyleConfig{ColorMode: ColorModeBright, DisableStyling: true})
	if err := terminal.WriteStyled(style, "plain"); err != nil {
		t.Fatal(err)
	}
	if err := terminal.Flush(); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); got != "plain" {
		t.Fatalf("disabled terminal output = %q, want plain text", got)
	}
}

func TestNoColorSuppressesColorsButCanPreserveAttributes(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	var output bytes.Buffer
	terminal := NewTerminal(&output, Size{Rows: 1, Columns: 8})
	if err := terminal.WriteStyled(Style{Foreground: ColorRed, Attributes: AttrBold}, "secret"); err != nil {
		t.Fatal(err)
	}
	if err := terminal.Flush(); err != nil {
		t.Fatal(err)
	}
	if got, want := output.String(), "\x1b[1msecret\x1b[0m"; got != want {
		t.Fatalf("NO_COLOR output = %q, want %q", got, want)
	}

	output.Reset()
	terminal = NewTerminalWithConfig(&output, Size{Rows: 1, Columns: 8}, StyleConfig{ColorMode: ColorModeANSI, HonorNoColor: false})
	if err := terminal.WriteStyled(Style{Foreground: ColorRed}, "red"); err != nil {
		t.Fatal(err)
	}
	if err := terminal.Flush(); err != nil {
		t.Fatal(err)
	}
	if got, want := output.String(), "\x1b[31mred\x1b[0m"; got != want {
		t.Fatalf("NO_COLOR override output = %q, want %q", got, want)
	}
}

func TestMaskSecretHandlesUnicodeEmptyAndANSILookingValues(t *testing.T) {
	if got, want := MaskSecret("secret"), "••••••"; got != want {
		t.Fatalf("masked ASCII = %q, want %q", got, want)
	}
	if got, want := MaskSecret("é界"), "•••"; got != want {
		t.Fatalf("masked Unicode = %q, want %q", got, want)
	}
	if got := MaskSecret(""); got != "" {
		t.Fatalf("masked empty = %q, want empty", got)
	}

	secret := "\x1b[31mTOP-SECRET"
	masked := MaskSecret(secret)
	if strings.Contains(masked, secret) || strings.Contains(masked, "TOP-SECRET") || strings.Contains(masked, "\x1b") {
		t.Fatalf("mask contains secret/control content: %q", masked)
	}

	frame := NewFrame(Size{Rows: 1, Columns: 12})
	frame.DrawMaskedText(0, 0, Style{}, secret)
	if got := frameRow(frame, 0); strings.Contains(got, "TOP-SECRET") || strings.Contains(got, "31m") {
		t.Fatalf("masked frame leaked input: %q", got)
	}

	var output bytes.Buffer
	terminal := NewTerminalWithConfig(&output, Size{Rows: 1, Columns: 20}, StyleConfig{ColorMode: ColorModeANSI})
	if err := terminal.WriteMasked(Style{Foreground: ColorRed}, secret); err != nil {
		t.Fatal(err)
	}
	if err := terminal.Flush(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "TOP-SECRET") || strings.Contains(output.String(), "\\x1b[31mTOP-SECRET") {
		t.Fatalf("masked terminal output leaked input: %q", output.String())
	}
}

func TestMaskedRenderingDoesNotPutSecretInRendererErrors(t *testing.T) {
	secret := "do-not-print-this-secret"
	terminal := NewTerminalWithConfig(&bytes.Buffer{}, Size{Rows: 1, Columns: 1}, StyleConfig{ColorMode: ColorModeNone})
	err := terminal.Render(nil)
	if err == nil || strings.Contains(err.Error(), secret) {
		t.Fatalf("renderer error leaked secret: %v", err)
	}
}

package typst

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"looz.ws/typstify/utils"
)

var (
	cmdBuilder utils.CmdBuilder
)

// use init function to setup PATH.
func SetupCmdBuilder(externalExe string) {
	exists, isDir := utils.CheckFileExists(externalExe)
	if exists && !isDir {
		cmdBuilder.Path = externalExe
	} else {
		exeName := "typst"
		if runtime.GOOS == "windows" {
			exeName = "typst.exe"
		}
		cmdBuilder.Path = exeName
	}

	cmdBuilder.DefaultArgs = []string{"--color=never"}
	cmdBuilder.Check()
}

func InitCmd(template string, dir string, opts *InitCmdOptions) error {
	args := []string{"init"}
	args = append(args, opts.Build()...)
	args = append(args, template, dir)

	cmd := cmdBuilder.Build(context.Background(), args...)

	//log.Println("command: ", cmd.String())

	out, err := cmd.Output()
	if len(out) > 0 {
		log.Println("typst init output: ")
		log.Println(string(out))
	}

	return err
}

// QueryEvalCmd runs `typst eval <expr> --in <file> --format json`, using the
// same root/font-path/package-path/features a compile of the same file would
// use, and returns the raw JSON stdout. This lets callers introspect the
// compiled document (e.g. real page positions via query()+location().page())
// through Typst's own stable CLI, instead of tinymist's undocumented internal
// preview protocol.
func QueryEvalCmd(ctx context.Context, opts *CompileCmdOptions, inputFile string, expr string) ([]byte, error) {
	args := []string{"eval", expr, "--in", inputFile, "--format", "json"}

	if opts.RootDir != "" {
		args = append(args, "--root", opts.RootDir)
	}
	for _, fontPath := range opts.FontPaths {
		if fontPath != "" {
			args = append(args, "--font-path", fontPath)
		}
	}
	if opts.PackagePath != "" {
		args = append(args, "--package-path", opts.PackagePath)
	}
	if opts.PackageCachePath != "" {
		args = append(args, "--package-cache-path", opts.PackageCachePath)
	}
	if opts.Features != "" {
		args = append(args, "--features", opts.Features)
	}
	for k, v := range opts.Input {
		args = append(args, fmt.Sprintf("--input=%s=%s", k, v))
	}

	cmd := cmdBuilder.Build(ctx, args...)
	log.Println("executing command: ", cmd.String())

	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("typst eval failed: %w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, err
	}
	return out, nil
}

// QueryHeadingPages returns the physical page number (1-based) of every
// heading in the document, in document order -- via query(heading) +
// location().page(), which gives the true physical page regardless of any
// display-numbering reset (unlike counter(page).at(...)).
func QueryHeadingPages(ctx context.Context, opts *CompileCmdOptions, inputFile string) ([]int, error) {
	out, err := QueryEvalCmd(ctx, opts, inputFile, "query(heading).map(h => h.location().page())")
	if err != nil {
		return nil, err
	}

	pages := make([]int, 0)
	if err := json.Unmarshal(out, &pages); err != nil {
		return nil, fmt.Errorf("parse heading pages: %w", err)
	}
	return pages, nil
}

type FontVariant struct {
	Style   string `json:"style"`
	Weight  int    `json:"weight"`
	Stretch string `json:"stretch"`
}

type FontFamily struct {
	Name     string        `json:"name"`
	Variants []FontVariant `json:"variants,omitempty"`
}

// FontCmd runs `typst fonts` command in the editor environment and return parsed
// font families.
//
// Command sample output:
//
//	Grantha Sangam MN
//	- Style: Normal, Weight: 400, Stretch: FontStretch(1000)
//	- Style: Normal, Weight: 700, Stretch: FontStretch(1000)
//	Gujarati MT
//	- Style: Normal, Weight: 400, Stretch: FontStretch(1000)
//	- Style: Normal, Weight: 700, Stretch: FontStretch(1000)
//	Gujarati Sangam MN
//	- Style: Normal, Weight: 400, Stretch: FontStretch(1000)
//	- Style: Normal, Weight: 700, Stretch: FontStretch(1000)
//
// If variants is not passed, no variant list is returned.
func FontsCmd(ctx context.Context, opts *FontCmdOptions) ([]FontFamily, error) {
	args := []string{"fonts"}
	args = append(args, opts.Build()...)

	cmd := cmdBuilder.Build(ctx, args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(out))

	var variantRE = regexp.MustCompile(
		`^- Style:\s*(.+?),\s*Weight:\s*(\d+),\s*Stretch:\s*(.+)$`,
	)

	fonts := make([]FontFamily, 0)
	var current *FontFamily

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if !strings.HasPrefix(line, "- ") {
			// start of a new font
			if current != nil {
				fonts = append(fonts, *current)
			}
			current = &FontFamily{Name: line}
			continue
		}

		if current == nil {
			return nil, fmt.Errorf("variant without family: %q", line)
		}

		// parse varient line
		m := variantRE.FindStringSubmatch(line)
		if m == nil {
			return nil, fmt.Errorf("invalid variant line: %q", line)
		}

		weight, err := strconv.Atoi(m[2])
		if err != nil {
			return nil, fmt.Errorf("invalid variant weight: %q", m[2])
		}

		current.Variants = append(current.Variants, FontVariant{
			Style:   m[1],
			Weight:  weight,
			Stretch: m[3],
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return fonts, nil
}

func VersionCmd() string {
	cmd := cmdBuilder.Build(context.Background(), "--version")
	out, _ := cmd.Output()

	pat := regexp.MustCompile(`^typst\s+(\S+)`)
	match := pat.FindSubmatch(out)
	if match == nil {
		return strings.TrimSpace(string(out))
	}

	return string(match[1])
}

var (
	version string
)

func CurrentVersion() string {
	if version == "" {
		version = VersionCmd()
	}

	return version
}

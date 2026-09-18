package view

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"

	"gioui.org/font"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/coder/acp-go-sdk"
	"github.com/oligo/gioview/misc"
	"github.com/oligo/gioview/theme"
	"github.com/oligo/gvcode"
	"github.com/oligo/gvcode/addons/completion"
	gvcolor "github.com/oligo/gvcode/color"
	"github.com/oligo/gvcode/textstyle/syntax"
	"github.com/sahilm/fuzzy"
	"golang.design/x/clipboard"

	"looz.ws/typstify/agent"
	"looz.ws/typstify/i18n"
	"looz.ws/typstify/widgets/icons"
)

var (
	clipInitOnce sync.Once
	clipInitErr  error
	xIcon        = icons.NewSvgIcon(icons.X)
)

func ensureClipboardInit() error {
	clipInitOnce.Do(func() {
		clipInitErr = clipboard.Init()
		if clipInitErr != nil {
			log.Printf("clipboard: init failed: %v", clipInitErr)
		}
	})
	return clipInitErr
}

// ImageAttachment represents an image attached in the input box or in chat history.
type ImageAttachment struct {
	Data      []byte
	MimeType  string
	Image     image.Image
	ImageOp   paint.ImageOp
	RemoveBtn widget.Clickable
}

type InputBox struct {
	rootDir string
	*gvcode.Editor
	colorScheme *syntax.ColorScheme
	cmdPopup    *completion.CompletionPopup
	rsPopup     *completion.CompletionPopup
	submit      bool

	attachments []*ImageAttachment
	mu          sync.Mutex
}

func newInputBox(session *agent.ACPSession) *InputBox {
	ed := &gvcode.Editor{}

	ed.WithOptions(
		gvcode.WrapLine(true),
		gvcode.WithSoftTab(true),
		gvcode.WithCornerRadius(unit.Dp(4)),
		gvcode.WithTabWidth(2),
	)

	cm := &completion.DefaultCompletion{Editor: ed}
	cmdCompletor := &commandCompletor{session: session}
	rsCompletor := &resourceCompletor{session: session}

	cmdPopup := completion.NewCompletionPopup(ed, cm)
	rsPopup := completion.NewCompletionPopup(ed, cm)
	cm.AddCompletor(rsCompletor, rsPopup)
	cm.AddCompletor(cmdCompletor, cmdPopup)

	ed.WithOptions(gvcode.WithAutoCompletion(cm))

	b := &InputBox{
		rootDir:  session.Cwd,
		Editor:   ed,
		cmdPopup: cmdPopup,
		rsPopup:  rsPopup,
	}

	ed.RegisterCommand("input-box",
		key.Filter{Name: key.NameEnter, Required: key.ModShift},
		func(gtx layout.Context, evt key.Event) gvcode.EditorEvent {
			b.submit = true
			return nil
		})

	ed.RegisterCommand("input-box",
		key.Filter{Name: key.NameReturn, Required: key.ModShift},
		func(gtx layout.Context, evt key.Event) gvcode.EditorEvent {
			b.submit = true
			return nil
		})

	ed.RegisterCommand("input-box",
		key.Filter{Name: "V", Required: key.ModShortcut},
		func(gtx layout.Context, evt key.Event) gvcode.EditorEvent {
			if err := ensureClipboardInit(); err == nil {
				imgData, err := clipboard.Read(context.Background(), clipboard.FmtImage)
				if err == nil && len(imgData) > 0 {
					if b.AddImageAttachment(imgData) {
						gtx.Execute(op.InvalidateCmd{})
						return nil
					}
				}
				textData, err := clipboard.Read(context.Background(), clipboard.FmtText)
				if err == nil && len(textData) > 0 {
					b.Editor.Insert(string(textData))
					return nil
				}
			}
			return nil
		})

	return b
}

func (b *InputBox) AddImageAttachment(data []byte) bool {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		log.Printf("inputbox: failed to decode image: %v", err)
		return false
	}

	mimeType := "image/png"
	if format != "" {
		mimeType = "image/" + format
	}

	b.mu.Lock()
	b.attachments = append(b.attachments, &ImageAttachment{
		Data:     data,
		MimeType: mimeType,
		Image:    img,
		ImageOp:  paint.NewImageOp(img),
	})
	b.mu.Unlock()
	return true
}

func (b *InputBox) Attachments() []ImageAttachment {
	b.mu.Lock()
	defer b.mu.Unlock()
	res := make([]ImageAttachment, len(b.attachments))
	for i, att := range b.attachments {
		res[i] = *att
	}
	return res
}

func (b *InputBox) HasAttachments() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.attachments) > 0
}

func (b *InputBox) ClearAttachments() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.attachments = nil
}

func (b *InputBox) Clear() {
	if b.Editor != nil {
		b.SetText("")
	}
	b.ClearAttachments()
}

func (b *InputBox) Update(gtx C) bool {
	for {
		_, ok := b.Editor.Update(gtx)
		if !ok {
			break
		}
	}

	submitted := b.submit
	b.submit = false
	return submitted
}

func (b *InputBox) layoutAttachments(gtx C, th *theme.Theme) D {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.attachments) == 0 {
		return D{}
	}

	var children []layout.FlexChild
	for i := 0; i < len(b.attachments); i++ {
		idx := i
		att := b.attachments[idx]

		if att.RemoveBtn.Clicked(gtx) {
			b.attachments = slices.Delete(b.attachments, idx, idx+1)
			return D{}
		}

		children = append(children, layout.Rigid(func(gtx C) D {
			return layout.Inset{Right: unit.Dp(8), Bottom: unit.Dp(4)}.Layout(gtx, func(gtx C) D {
				thumbSize := image.Pt(gtx.Dp(unit.Dp(56)), gtx.Dp(unit.Dp(56)))
				rr := gtx.Dp(unit.Dp(4))
				rect := image.Rectangle{Max: thumbSize}

				return layout.Stack{}.Layout(gtx,
					layout.Stacked(func(gtx C) D {
						defer clip.UniformRRect(rect, rr).Push(gtx.Ops).Pop()
						paint.Fill(gtx.Ops, misc.WithAlpha(th.ContrastBg, 0x18))

						im := widget.Image{
							Src: att.ImageOp,
							Fit: widget.Cover,
						}
						gtx.Constraints = layout.Exact(thumbSize)
						dims := im.Layout(gtx)

						// Border
						defer clip.Stroke{
							Path:  clip.UniformRRect(rect, rr).Path(gtx.Ops),
							Width: float32(gtx.Dp(unit.Dp(1))),
						}.Op().Push(gtx.Ops).Pop()
						paint.Fill(gtx.Ops, misc.WithAlpha(th.Fg, 0x40))

						return dims
					}),
					layout.Stacked(func(gtx C) D {
						btnSize := gtx.Dp(unit.Dp(16))
						return layout.Inset{
							Left: unit.Dp(float32(thumbSize.X-btnSize) / gtx.Metric.PxPerDp),
						}.Layout(gtx, func(gtx C) D {
							gtx.Constraints = layout.Exact(image.Pt(btnSize, btnSize))
							return material.Clickable(gtx, &att.RemoveBtn, func(gtx C) D {
								delRect := image.Rectangle{Max: image.Pt(btnSize, btnSize)}
								defer clip.UniformRRect(delRect, btnSize/2).Push(gtx.Ops).Pop()
								paint.Fill(gtx.Ops, th.Bg)
								return xIcon.Layout(gtx, th.Fg, unit.Sp(10))
							})
						})
					}),
				)
			})
		}))
	}

	return layout.Inset{Left: unit.Dp(6), Top: unit.Dp(4)}.Layout(gtx, func(gtx C) D {
		return layout.Flex{
			Axis:      layout.Horizontal,
			Alignment: layout.Middle,
		}.Layout(gtx, children...)
	})
}

func (b *InputBox) Layout(gtx C, th *theme.Theme) D {
	cs := syntax.ColorScheme{}
	cs.Background = gvcolor.MakeColor(th.Bg)
	cs.Foreground = gvcolor.MakeColor(th.Fg)
	cs.SelectColor = gvcolor.MakeColor(th.ContrastBg).MulAlpha(th.SelectedAlpha)
	b.Editor.WithOptions(
		gvcode.WithFont(font.Font{Typeface: th.Face}),
		gvcode.WithTextSize(th.TextSize),
		gvcode.WithTextAlignment(text.Start),
		gvcode.WithLineHeight(0, 1.5),
		gvcode.WithColorScheme(cs),
	)

	gtx.Constraints.Max.Y = min(gtx.Dp(unit.Dp(160)), gtx.Constraints.Max.Y)
	popupSize := image.Point{
		X: int(float32(gtx.Constraints.Max.X) * 0.8),
		Y: int(float32(gtx.Constraints.Max.Y) * 0.9),
	}
	b.cmdPopup.Theme = th.Theme
	b.cmdPopup.Size = popupSize
	b.cmdPopup.TextSize = th.TextSize
	b.rsPopup.Theme = th.Theme
	b.rsPopup.Size = popupSize
	b.rsPopup.TextSize = th.TextSize

	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx C) D {
			return b.layoutAttachments(gtx, th)
		}),
		layout.Rigid(func(gtx C) D {
			return layout.UniformInset(unit.Dp(6)).Layout(gtx, func(gtx C) D {
				macro := op.Record(gtx.Ops)
				label := material.Label(th.Theme, th.TextSize, i18n.Translate("Type @ to include resource, and / to use skills."))
				label.Color = misc.WithAlpha(th.Fg, 0xb0)
				_ = label.Layout(gtx)
				call := macro.Stop()

				editorDims := b.Editor.Layout(gtx, th.Shaper)

				if b.Editor.Text() == "" {
					call.Add(gtx.Ops)
				}

				return editorDims
			})
		}),
	)
}

var resourceMentionPattern = regexp.MustCompile(`(?:^|\s)@([\w\.\-\/]+)(?:\s|$)`)

// Parse user input to extract text, attached images, and resource-mentions into ACP content blocks.
func (b *InputBox) Blocks() []acp.ContentBlock {
	var text string
	if b.Editor != nil {
		text = strings.TrimSpace(b.Text())
	}
	b.mu.Lock()
	attachments := make([]*ImageAttachment, len(b.attachments))
	copy(attachments, b.attachments)
	b.mu.Unlock()

	if len(text) == 0 && len(attachments) == 0 {
		return nil
	}

	blocks := make([]acp.ContentBlock, 0)

	if len(text) > 0 {
		blocks = append(blocks, acp.TextBlock(text))
	}

	for _, att := range attachments {
		b64 := base64.StdEncoding.EncodeToString(att.Data)
		blocks = append(blocks, acp.ImageBlock(b64, att.MimeType))
	}

	if len(text) > 0 {
		resourceMatches := resourceMentionPattern.FindAllStringSubmatch(text, -1)
		if len(resourceMatches) > 0 {
			resourceMap := make(map[string]bool)

			for _, match := range resourceMatches {
				if len(match) <= 1 {
					continue
				}

				resPath := match[1] // the capture group
				absResPath, err := filepath.Abs(filepath.Join(b.rootDir, resPath))
				if err != nil {
					continue
				}
				if _, exists := resourceMap[absResPath]; exists {
					continue
				}

				blocks = append(blocks, acp.ResourceLinkBlock(resPath, fmt.Sprintf("file://%s", absResPath)))
				resourceMap[absResPath] = true
			}
		}
	}

	return blocks
}

var _ gvcode.Completor = (*commandCompletor)(nil)
var _ gvcode.Completor = (*resourceCompletor)(nil)

type commandCompletor struct {
	session *agent.ACPSession
}

func (c *commandCompletor) Trigger() gvcode.Trigger {
	return gvcode.Trigger{
		Characters: []string{"/"},
		Policy:     completion.ExplicitTriggerPolicy{},
	}
}

func (c *commandCompletor) Suggest(ctx gvcode.CompletionContext) []gvcode.CompletionCandidate {
	if c.session == nil {
		return nil
	}
	commands := c.session.AvailableCommands()
	candidates := make([]gvcode.CompletionCandidate, 0, len(commands))
	for _, cmd := range commands {
		candidates = append(candidates, gvcode.CompletionCandidate{
			Label:       "/" + cmd.Name,
			TextEdit:    gvcode.TextEdit{NewText: cmd.Name},
			Description: cmd.Description,
			Kind:        "snippet",
		})
	}
	return candidates
}

func (c *commandCompletor) FilterAndRank(pattern string, candidates []gvcode.CompletionCandidate) []gvcode.CompletionCandidate {
	if pattern == "" {
		return candidates
	}
	filtered := candidates[:0]
	lower := strings.ToLower(pattern)
	for _, cand := range candidates {
		if strings.Contains(strings.ToLower(cand.Label), lower) {
			filtered = append(filtered, cand)
		}
	}
	return filtered
}

type resourceCompletor struct {
	session *agent.ACPSession
}

func (r *resourceCompletor) Trigger() gvcode.Trigger {
	return gvcode.Trigger{
		Characters: []string{"@"},
		Policy:     completion.ExplicitTriggerPolicy{},
	}
}

func (r *resourceCompletor) Suggest(ctx gvcode.CompletionContext) []gvcode.CompletionCandidate {
	if r.session == nil {
		return nil
	}

	root := r.session.Cwd
	candidates := []gvcode.CompletionCandidate{}

	err := filepath.WalkDir(root, func(path string, e os.DirEntry, err error) error {
		if err != nil || path == root {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
			return nil
		}

		resourcePath := filepath.ToSlash(rel)
		kind := "file"
		if e.IsDir() {
			kind = "folder"
			resourcePath += "/"
		}
		candidates = append(candidates, gvcode.CompletionCandidate{
			Label:    resourcePath,
			TextEdit: gvcode.TextEdit{NewText: resourcePath},
			Kind:     kind,
		})
		return nil
	})
	if err != nil {
		return nil
	}
	return candidates
}

type resourceCandidateSource struct {
	candidates []gvcode.CompletionCandidate
}

func (src *resourceCandidateSource) String(i int) string {
	return src.candidates[i].Label
}

func (src *resourceCandidateSource) Len() int {
	return len(src.candidates)
}

func (r *resourceCompletor) FilterAndRank(pattern string, candidates []gvcode.CompletionCandidate) []gvcode.CompletionCandidate {
	if pattern == "" {
		return candidates
	}

	pattern = strings.ToLower(strings.TrimLeft(strings.ReplaceAll(pattern, "\\", "/"), "/"))
	if pattern == "" || strings.Contains(pattern, "..") {
		return nil
	}

	matches := fuzzy.FindFrom(pattern, &resourceCandidateSource{candidates: candidates})
	filtered := make([]gvcode.CompletionCandidate, 0, len(matches))
	for _, match := range matches {
		filtered = append(filtered, candidates[match.Index])
	}
	return filtered
}

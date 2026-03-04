package emojix

import (
	"emojix/model"
	"html/template"
	"io"
)

type IndexPageViewParam struct {
	Title    string
	Nickname string
}

type GamePageViewParam struct {
	GameID      string
	Leaderboard []model.LeaderboardEntry
	Messages    []model.GameStateMessage

	MaskedWord []string
	EmojiHint  string
}

type GameLoadingPageViewParam struct {
	GameID string
}

type GameWordViewParam struct {
	MaskedWord []string
}

type GameLeaderboardViewParam struct {
	Leaderboard []model.LeaderboardEntry
}

type GameMsgViewParam = model.GameStateMessage

type View interface {
	renderErrorPage(wr io.Writer) error

	renderIndexPage(wr io.Writer, params IndexPageViewParam) error

	renderGamePage(wr io.Writer, params GamePageViewParam) error
	renderGameWord(wr io.Writer, params GameWordViewParam) error
	renderGameMsg(wr io.Writer, params GameMsgViewParam) error
	renderGameLeaderboard(wr io.Writer, params GameLeaderboardViewParam) error

	renderGameLoadingPage(wr io.Writer, params GameLoadingPageViewParam) error
}

type htmlView struct {
	indexPageTemplate       template.Template
	gamePageTemplate        template.Template
	gameWordTemplate        template.Template
	gameMsgTemplate         template.Template
	gameLeaderboardTemplate template.Template
	gameLoadingPageTemplate template.Template
	errorPageTemplate       template.Template
}

func NewHTMLView() View {
	indexPageTemplate := *template.Must(template.ParseFiles(
		"template/base.gohtml",
		"template/index.gohtml",
	))

	gamePageTemplate := *template.Must(template.ParseFiles(
		"template/base.gohtml",
		"template/game.gohtml",
		"template/game-msg-def.gohtml",
		"template/game-leaderboard-def.gohtml",
		"template/game-word-def.gohtml",
	))
	gameWordTemplate := *template.Must(template.ParseFiles(
		"template/game-word.gohtml",
		"template/game-word-def.gohtml",
	))
	gameMsgTemplate := *template.Must(template.ParseFiles(
		"template/game-msg.gohtml",
		"template/game-msg-def.gohtml",
	))
	gameLeaderboardTemplate := *template.Must(template.ParseFiles(
		"template/game-leaderboard.gohtml",
		"template/game-leaderboard-def.gohtml",
	))

	gameLoadingPageTemplate := *template.Must(template.ParseFiles(
		"template/base.gohtml",
		"template/game-loading.gohtml",
	))

	errorPageTemplate := *template.Must(template.ParseFiles(
		"template/base.gohtml",
		"template/error.gohtml",
	))

	return &htmlView{
		indexPageTemplate:       indexPageTemplate,
		gamePageTemplate:        gamePageTemplate,
		gameWordTemplate:        gameWordTemplate,
		gameMsgTemplate:         gameMsgTemplate,
		gameLeaderboardTemplate: gameLeaderboardTemplate,
		gameLoadingPageTemplate: gameLoadingPageTemplate,
		errorPageTemplate:       errorPageTemplate,
	}
}

func (v *htmlView) renderIndexPage(wr io.Writer, params IndexPageViewParam) error {
	return v.indexPageTemplate.Execute(wr, params)
}

func (v *htmlView) renderGamePage(wr io.Writer, params GamePageViewParam) error {
	return v.gamePageTemplate.Execute(wr, params)
}

func (v *htmlView) renderGameLeaderboard(wr io.Writer, params GameLeaderboardViewParam) error {
	return v.gameLeaderboardTemplate.Execute(wr, params)
}

func (v *htmlView) renderGameMsg(wr io.Writer, params GameMsgViewParam) error {
	return v.gameMsgTemplate.Execute(wr, params)
}

func (v *htmlView) renderGameWord(wr io.Writer, params GameWordViewParam) error {
	return v.gameWordTemplate.Execute(wr, params)
}
func (v *htmlView) renderGameLoadingPage(wr io.Writer, params GameLoadingPageViewParam) error {
	return v.gameLoadingPageTemplate.Execute(wr, params)
}

func (v *htmlView) renderErrorPage(wr io.Writer) error {
	return v.errorPageTemplate.Execute(wr, nil)
}

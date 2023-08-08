package renderer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/obalunenko/telegram-ride-announcer-bot/internal/service/renderer"
)

type TemplatesSuite struct {
	suite.Suite
	tpls renderer.Renderer
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestExampleTestSuite(t *testing.T) {
	suite.Run(t, new(TemplatesSuite))
}

func (s *TemplatesSuite) loadGoldenFile(name string) string {
	file, err := os.ReadFile(filepath.Join("testdata", name))
	s.Require().NoError(err)

	return string(file)
}

func (s *TemplatesSuite) SetupSuite() {
	var err error

	s.tpls, err = renderer.New()
	s.Require().NoError(err)
	s.Require().NotNil(s.tpls)
}

func (s *TemplatesSuite) TestTemplates_Help() {
	params := renderer.HelpParams{
		BotUsername: "BotName",
		Commands:    "some commands",
		HelpCmd:     "/help",
	}

	res, err := s.tpls.Help(params)
	s.Assert().NoError(err)

	s.Assert().Equal(s.loadGoldenFile("help.golden"), res)
}

func (s *TemplatesSuite) TestTemplates_Welcome() {
	params := renderer.WelcomeParams{
		Firstname:   "Firstname",
		BotUsername: "BotName",
		HelpCmd:     "/help",
	}

	res, err := s.tpls.Welcome(params)
	s.Assert().NoError(err)

	s.Assert().Equal(s.loadGoldenFile("welcome.golden"), res)
}

func (s *TemplatesSuite) TestTemplates_Trip() {
	params := renderer.TripParams{
		Title:       "Title",
		Description: "Description",
		Date:        "Date",
		CreatedBy:   "CreatedBy",
	}

	res, err := s.tpls.Trip(params)
	s.Assert().NoError(err)

	s.Assert().Equal(s.loadGoldenFile("trip.golden"), res)
}

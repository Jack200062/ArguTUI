package loginscreen

import (
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	LoginScreenName = "login"
)

// Credentials holds the username and password
type Credentials struct {
	Username string
	Password string
}

// LoginScreen provides a form for username/password login
type LoginScreen struct {
	app      *tview.Application
	form     *tview.Form
	flex     *tview.Flex
	errorBox *tview.TextView
	onSubmit func(Credentials)
	onCancel func()

	mu          sync.Mutex
	credentials Credentials
	submitted   bool
}

// NewLoginScreen creates a new login screen
func NewLoginScreen(
	app *tview.Application,
	onSubmit func(Credentials),
	onCancel func(),
) *LoginScreen {
	return &LoginScreen{
		app:      app,
		onSubmit: onSubmit,
		onCancel: onCancel,
	}
}

func (s *LoginScreen) Name() string {
	return LoginScreenName
}

func (s *LoginScreen) Init() tview.Primitive {
	// Color scheme
	textColor := tcell.NewHexColor(0x00bebe)
	mainTextColor := tcell.NewHexColor(0x805700)
	backgroundColor := tcell.NewHexColor(0x000000)
	borderColor := tcell.NewHexColor(0x63a0bf)
	fieldBgColor := tcell.NewHexColor(0x1a1a1a)
	buttonBgColor := tcell.NewHexColor(0x017be9)
	buttonTextColor := tcell.NewHexColor(0xffffff)
	errorColor := tcell.NewHexColor(0xff5555)

	// Create error message box (hidden by default)
	s.errorBox = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetTextColor(errorColor)
	s.errorBox.SetBackgroundColor(backgroundColor)

	// Create form
	s.form = tview.NewForm().
		SetFieldBackgroundColor(fieldBgColor).
		SetFieldTextColor(mainTextColor).
		SetLabelColor(textColor).
		SetButtonBackgroundColor(buttonBgColor).
		SetButtonTextColor(buttonTextColor)

	s.form.
		SetBackgroundColor(backgroundColor).
		SetBorderColor(borderColor).
		SetBorder(true).
		SetTitle(" Login to ArgoCD ").
		SetTitleAlign(tview.AlignCenter).
		SetTitleColor(textColor)

	// Add form fields
	s.form.AddInputField("Username", "", 30, nil, func(text string) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.credentials.Username = text
	})

	s.form.AddPasswordField("Password", "", 30, '*', func(text string) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.credentials.Password = text
	})

	// Add buttons
	s.form.AddButton("Login", func() {
		s.mu.Lock()
		creds := s.credentials
		s.submitted = true
		s.mu.Unlock()

		if s.onSubmit != nil {
			s.onSubmit(creds)
		}
	})

	s.form.AddButton("Cancel", func() {
		if s.onCancel != nil {
			s.onCancel()
		}
	})

	// Set button colors
	s.form.SetButtonsAlign(tview.AlignCenter)

	// Handle keyboard shortcuts
	s.form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			if s.onCancel != nil {
				s.onCancel()
			}
			return nil
		case tcell.KeyCtrlC:
			s.app.Stop()
			return nil
		}
		return event
	})

	// Create a container for form and error message
	formContainer := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(s.form, 0, 1, true).
		AddItem(s.errorBox, 2, 0, false)

	// Create flex container to center the form
	s.flex = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(
			tview.NewFlex().
				SetDirection(tview.FlexColumn).
				AddItem(nil, 0, 1, false).
				AddItem(formContainer, 50, 1, true).
				AddItem(nil, 0, 1, false),
			14, 1, true,
		).
		AddItem(nil, 0, 1, false)

	s.flex.SetBackgroundColor(backgroundColor)

	return s.flex
}

// GetCredentials returns the submitted credentials
func (s *LoginScreen) GetCredentials() (Credentials, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.credentials, s.submitted
}

// ShowError displays an error message on the login screen
func (s *LoginScreen) ShowError(message string) {
	if s.errorBox != nil {
		s.errorBox.SetText(message)
	}
}

// ClearError clears the error message
func (s *LoginScreen) ClearError() {
	if s.errorBox != nil {
		s.errorBox.SetText("")
	}
}

// Reset clears the form data
func (s *LoginScreen) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.credentials = Credentials{}
	s.submitted = false
}

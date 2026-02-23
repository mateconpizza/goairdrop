// Package notify.
package notify

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const command = "notify-send"

type State struct {
	LastID uint32 `json:"last_id"`
}

// Invalidate notification stream.
func (s *State) Invalidate() {
	s.LastID = 0
}

func (s *State) Send(m Mode, opts ...Option) error {
	if m == Disabled {
		return nil
	}

	nt := New(opts...)

	if s.LastID != 0 {
		nt.ID = s.LastID
	}

	id, err := nt.Send()
	if err != nil {
		return err
	}

	s.LastID = id
	return nil
}

type Mode int

const (
	Disabled Mode = iota
	OnActive
	OnInactive
	OnCritical
	OnError
	OnChange // active ↔ inactive
)

// Urgency levels for notifications.
type Urgency string

const (
	UrgencyLow      Urgency = "low"
	UrgencyCritical Urgency = "critical"
	UrgencyNormal   Urgency = "normal"
)

const (
	// Dialog.
	IconInfo     = "dialog-information"
	IconWarning  = "dialog-warning"
	IconError    = "dialog-error"
	IconQuestion = "dialog-question"

	// Misc.
	IconNetwork         = "network-idle"
	IconBattery         = "battery"
	IconBatteryLow      = "battery-low-symbolic"
	IconBatteryCritical = "battery-000-symbolic"
	IconAudio           = "audio-card"
	IconFolder          = "folder"
	IconSystem          = "system"
	IconKeyboard        = "keyboard"
)

// Category represents a notification category as defined by the spec.
type Category string

const (
	// Call categories.
	CategoryCall           Category = "call"
	CategoryCallEnded      Category = "call.ended"
	CategoryCallIncoming   Category = "call.incoming"
	CategoryCallUnanswered Category = "call.unanswered"

	// Device categories.
	CategoryDevice        Category = "device"
	CategoryDeviceAdded   Category = "device.added"
	CategoryDeviceError   Category = "device.error"
	CategoryDeviceRemoved Category = "device.removed"

	// Email categories.
	CategoryEmail        Category = "email"
	CategoryEmailArrived Category = "email.arrived"
	CategoryEmailBounced Category = "email.bounced"

	// Instant messaging categories.
	CategoryIM         Category = "im"
	CategoryIMError    Category = "im.error"
	CategoryIMReceived Category = "im.received"

	// Network categories.
	CategoryNetwork             Category = "network"
	CategoryNetworkConnected    Category = "network.connected"
	CategoryNetworkDisconnected Category = "network.disconnected"
	CategoryNetworkError        Category = "network.error"

	// Presence categories.
	CategoryPresence        Category = "presence"
	CategoryPresenceOffline Category = "presence.offline"
	CategoryPresenceOnline  Category = "presence.online"

	// Transfer categories.
	CategoryTransfer         Category = "transfer"
	CategoryTransferComplete Category = "transfer.complete"
	CategoryTransferError    Category = "transfer.error"
)

type Option func(*Notification)

// Notification represents a desktop notification.
type Notification struct {
	Summary  string        // Notification title
	Body     string        // Notification message/body
	Icon     string        // Icon name or path (e.g., "dialog-information", "/path/to/icon.png")
	Urgency  Urgency       // Urgency level (low, normal, critical)
	Category Category      // Category for grouping (optional)
	AppName  string        // Application name (optional)
	Timeout  time.Duration // How long to show notification (0 = default)
	ID       uint32        // Notification ID (use for replacing current notification)
	ctx      context.Context
}

func WithSummary(s string) Option {
	return func(n *Notification) {
		n.Summary = s
	}
}

func WithBody(b string) Option {
	return func(n *Notification) {
		n.Body = b
	}
}

func WithAppName(a string) Option {
	return func(n *Notification) {
		n.AppName = a
	}
}

func WithIcon(i string) Option {
	return func(n *Notification) {
		n.Icon = i
	}
}

func WithContext(ctx context.Context) Option {
	return func(n *Notification) {
		n.ctx = ctx
	}
}

func WithUrgency(u Urgency) Option {
	return func(n *Notification) {
		n.Urgency = u
	}
}

func WithCategory(c Category) Option {
	return func(n *Notification) {
		n.Category = c
	}
}

func WithTimeout(t time.Duration) Option {
	return func(n *Notification) {
		n.Timeout = t
	}
}

func WithID(id uint32) Option {
	return func(n *Notification) {
		n.ID = id
	}
}

// Send sends a desktop notification using notify-send.
func (nt *Notification) Send() (uint32, error) {
	args := buildNotifyArgs(nt)
	args = append(args, "--print-id")

	cmd := exec.CommandContext(nt.ctx, command, args...)
	cmd.Env = os.Environ()

	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	id, err := strconv.ParseUint(strings.TrimSpace(string(out)), 10, 32)
	if err != nil {
		return 0, err
	}

	return uint32(id), nil
}

func (nt *Notification) Close() error {
	return Close(nt.ctx, int(nt.ID))
}

func Close(ctx context.Context, id int) error {
	cmd := exec.CommandContext(ctx, "dunstctl", "close", strconv.Itoa(id))
	cmd.Env = os.Environ()
	return cmd.Run()
}

func New(opts ...Option) *Notification {
	n := &Notification{}
	for _, opt := range opts {
		opt(n)
	}

	if n.Summary == "" {
		n.Summary = "Notification"
	}

	if n.ctx == nil {
		n.ctx = context.Background()
	}

	return n
}

// Send sends a desktop notification using notify-send.
func Send(ctx context.Context, notif *Notification) (uint32, error) {
	args := buildNotifyArgs(notif)
	args = append(args, "--print-id")

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Env = os.Environ()

	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	id, err := strconv.ParseUint(strings.TrimSpace(string(out)), 10, 32)
	if err != nil {
		return 0, err
	}

	return uint32(id), nil
}

// // SendSimple sends a simple notification with just title and message.
// func SendSimple(ctx context.Context, summary, body string) (uint32, error) {
// 	return Send(ctx, &Notification{
// 		ReplaceID: 999,
// 		Summary:   summary,
// 		Body:      body,
// 		Icon:      IconInfo,
// 	})
// }
//
// // SendWithIcon sends a notification with an icon.
// func SendWithIcon(ctx context.Context, summary, body, icon string) (uint32, error) {
// 	return Send(ctx, &Notification{
// 		Summary: summary,
// 		Body:    body,
// 		Icon:    icon})
// }
//
// // SendUrgent sends a critical urgency notification.
// func SendUrgent(ctx context.Context, summary, body, icon string) (uint32, error) {
// 	return Send(ctx, &Notification{
// 		Summary: summary,
// 		Body:    body,
// 		Icon:    icon,
// 		Urgency: UrgencyCritical,
// 	})
// }

// buildNotifyArgs constructs the notify-send command arguments.
func buildNotifyArgs(nt *Notification) []string {
	args := make([]string, 0)

	// Add urgency
	if nt.Urgency != "" {
		args = append(args, "-u", string(nt.Urgency))
	}

	// Add icon
	if nt.Icon != "" {
		args = append(args, "-i", nt.Icon)
	}

	// Add timeout (in milliseconds)
	if nt.Timeout > 0 {
		ms := int(nt.Timeout.Milliseconds())
		args = append(args, "-t", strconv.Itoa(ms))
	}

	// Add category
	if nt.Category != "" {
		args = append(args, "-c", string(nt.Category))
	}

	// Add app name
	if nt.AppName != "" {
		args = append(args, "-a", nt.AppName)
	}

	// Add title and message
	args = append(args, nt.Summary)
	if nt.Body != "" {
		args = append(args, nt.Body)
	}

	if nt.ID != 0 {
		args = append(args, "-r", strconv.Itoa(int(nt.ID)))
	}

	return args
}

// IsAvailable checks if notify-send is available on the system.
func IsAvailable() bool {
	cmd := exec.Command("which", command)
	err := cmd.Run()
	return err == nil
}

// NotificationBuilder provides a fluent interface for building notifications.
type NotificationBuilder struct {
	notif *Notification
}

// Message sets the notification message.
func (nb *NotificationBuilder) Message(msg string) *NotificationBuilder {
	nb.notif.Body = msg
	return nb
}

// Icon sets the notification icon.
func (nb *NotificationBuilder) Icon(icon string) *NotificationBuilder {
	nb.notif.Icon = icon
	return nb
}

// Urgency sets the notification urgency.
func (nb *NotificationBuilder) Urgency(urgency Urgency) *NotificationBuilder {
	nb.notif.Urgency = urgency
	return nb
}

// Timeout sets how long to show the notification.
func (nb *NotificationBuilder) Timeout(timeout time.Duration) *NotificationBuilder {
	nb.notif.Timeout = timeout
	return nb
}

// Category sets the notification category.
func (nb *NotificationBuilder) Category(category Category) *NotificationBuilder {
	nb.notif.Category = category
	return nb
}

func (nb *NotificationBuilder) WithID() *NotificationBuilder {
	return nb
}

// AppName sets the application name.
func (nb *NotificationBuilder) AppName(appName string) *NotificationBuilder {
	nb.notif.AppName = appName
	return nb
}

// Send sends the notification.
func (nb *NotificationBuilder) Send(ctx context.Context) (uint32, error) {
	return Send(ctx, nb.notif)
}

// Build returns the constructed notification.
func (nb *NotificationBuilder) Build() *Notification {
	return nb.notif
}

// NotificationQueue manages queued notifications to avoid spam.
type NotificationQueue struct {
	interval time.Duration
	lastSent time.Time
}

// NewQueue creates a new notification queue with minimum interval between sends.
func NewQueue(minInterval time.Duration) *NotificationQueue {
	return &NotificationQueue{
		interval: minInterval,
	}
}

// Send sends a notification respecting the minimum interval.
func (nq *NotificationQueue) Send(ctx context.Context, notif *Notification) error {
	now := time.Now()
	if now.Sub(nq.lastSent) < nq.interval {
		// Too soon, skip this notification
		return nil
	}

	_, err := Send(ctx, notif)
	if err == nil {
		nq.lastSent = now
	}
	return err
}

// FormatList formats a slice of strings into a bulleted list for notifications.
func FormatList(items []string) string {
	var b strings.Builder
	for _, item := range items {
		b.WriteString("• ")
		b.WriteString(item)
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

// EscapeMarkup escapes special characters for Pango markup (used by some notification daemons).
func EscapeMarkup(text string) string {
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")
	return text
}

func Notify(ctx context.Context,
	summary, body string,
	opts ...Option,
) (uint32, error) {
	n := New(
		WithSummary(summary),
		WithBody(body),
		WithIcon(IconInfo),
		WithUrgency(UrgencyNormal),
		WithAppName("dwmtray"),
	)

	for _, opt := range opts {
		opt(n)
	}

	return Send(ctx, n)
}

func MaybeSend(
	ctx context.Context,
	mode Mode,
	prev, curr bool,
	summary, body string,
	opts ...Option,
) {
	switch mode {
	case OnInactive:
		if prev && !curr {
			Notify(ctx, summary, body,
				append(opts, WithUrgency(UrgencyCritical))...,
			)
		}
	case OnActive:
		if !prev && curr {
			Notify(ctx, summary, body, opts...)
		}
	case OnChange:
		if prev != curr {
			Notify(ctx, summary, body, opts...)
		}
	}
}

// Reference: https://specifications.freedesktop.org/notification/latest/categories.html

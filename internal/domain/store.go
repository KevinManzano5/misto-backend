package domain

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Store struct represents system user
type Store struct {
	ID     string
	UserID string
	Name   string
	// Slug is a URL-friendly, lowercase, hyphenated identifier derived from Name, exposed to the
	// frontend for building friendly URLs. It is NOT used for backend lookups (those stay
	// UUID-only) and is recomputed whenever Name changes.
	Slug                string
	Description         string
	IsActive            bool
	AppointmentDuration int32
	// Timezone is the store's IANA timezone (e.g. "America/Bogota"), used to interpret its
	// schedule's open/close hours and to resolve which calendar day an appointment falls on.
	Timezone string
	// AutoConfirmAppointments controls whether a new appointment is confirmed immediately or
	// left pending for the owner to approve/reject.
	AutoConfirmAppointments bool
	// Email is the store's contact email address.
	Email string
	// Address is the store's physical address.
	Address string
	// AllowSameDayAppointments controls whether a client can book an appointment on the same
	// calendar day it is requested, in the store's own timezone.
	AllowSameDayAppointments bool
	// PhoneNumber is the store's contact phone number, in E.164 format (e.g. "+573001234567").
	PhoneNumber string
	// CashOnSiteEnabled controls whether clients can choose to pay in cash at the store when
	// booking an appointment.
	CashOnSiteEnabled bool
	// CardAdvanceEnabled controls whether clients can choose to pay by card in advance when
	// booking an appointment.
	CardAdvanceEnabled bool
	// AppUsageFeePercentage is the percentage (0-100) the platform charges the store for app
	// usage. Visible and editable only by platform administrators — never exposed via the store
	// owner's own update/response flow.
	AppUsageFeePercentage int32
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// NewStore function
func NewStore(userID, name, slug, description string, isActive bool, appointmentDuration int32, timezone string, autoConfirmAppointments bool, email, address string, allowSameDayAppointments bool, phoneNumber string, cashOnSiteEnabled, cardAdvanceEnabled bool) *Store {
	now := time.Now()

	return &Store{
		UserID:                   userID,
		Name:                     name,
		Slug:                     slug,
		Description:              description,
		IsActive:                 isActive,
		AppointmentDuration:      appointmentDuration,
		Timezone:                 timezone,
		AutoConfirmAppointments:  autoConfirmAppointments,
		Email:                    email,
		Address:                  address,
		AllowSameDayAppointments: allowSameDayAppointments,
		PhoneNumber:              phoneNumber,
		CashOnSiteEnabled:        cashOnSiteEnabled,
		CardAdvanceEnabled:       cardAdvanceEnabled,
		CreatedAt:                now,
		UpdatedAt:                now,
	}
}

// IsValid method
func (store *Store) IsValid() bool {
	return store.UserID != "" && store.Name != "" && store.AppointmentDuration > 0 && store.Timezone != ""
}

// maxSlugLength caps the generated slug at the same width as the name column (VARCHAR(100)).
const maxSlugLength = 100

// SlugFromName derives a URL-friendly slug from a store name: lowercased, diacritics stripped
// (e.g. "Café" -> "cafe"), any run of characters outside [a-z0-9] collapsed to a single '-',
// with leading/trailing '-' trimmed. If the name contains no [a-z0-9] characters at all (e.g.
// an all-symbol/emoji name, which passes the "min=2 runes" validation on name), it falls back
// to a short random identifier so a Store never ends up with an empty slug.
func SlugFromName(name string) string {
	decomposed := norm.NFD.String(name)

	var builder strings.Builder
	builder.Grow(len(decomposed))

	lastWasHyphen := false
	for _, r := range decomposed {
		switch {
		case unicode.Is(unicode.Mn, r):
			// Combining mark produced by NFD decomposition (e.g. the accent on "é"); drop it,
			// keeping the base letter already written.
			continue
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastWasHyphen = false
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r + ('a' - 'A'))
			lastWasHyphen = false
		default:
			if !lastWasHyphen && builder.Len() > 0 {
				builder.WriteByte('-')
				lastWasHyphen = true
			}
		}
	}

	slug := strings.TrimRight(builder.String(), "-")

	if len(slug) > maxSlugLength {
		slug = strings.TrimRight(slug[:maxSlugLength], "-")
	}

	if slug == "" {
		return "store-" + randomSlugSuffix()
	}

	return slug
}

// ApplyPercentageFee adds a percentage fee on top of amount, rounded to the nearest whole unit
// (round-half-up). Used for COP amounts, which have no decimal subunit, so the fee itself must
// be a whole number of pesos.
func ApplyPercentageFee(amount int64, feePercentage int32) int64 {
	fee := (amount*int64(feePercentage) + 50) / 100
	return amount + fee
}

// randomSlugSuffix returns an 8 hex-character random suffix used as a fallback slug for names
// with no [a-z0-9] characters at all.
func randomSlugSuffix() string {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand.Read on the standard reader does not fail in practice; fall back to a
		// time-derived value rather than panicking.
		return hex.EncodeToString([]byte(time.Now().Format("15040502")))
	}
	return hex.EncodeToString(buf)
}

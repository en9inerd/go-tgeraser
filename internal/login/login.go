package login

import (
	"context"
	"errors"
	"fmt"

	"github.com/en9inerd/go-pkgs/promptio"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

const maxAuthAttempts = 3

// Authorize logs in interactively unless the session is already authorized.
func Authorize(ctx context.Context, api *auth.Client) error {
	status, err := api.Status(ctx)
	if err != nil {
		return fmt.Errorf("check auth status: %w", err)
	}
	if status.Authorized {
		return nil
	}

	phone, err := promptio.ReadLine(ctx, "Enter your phone number (international format, e.g. +12345678900): ")
	if err != nil {
		return err
	}

	sent, err := api.SendCode(ctx, phone, auth.SendCodeOptions{})
	if tg.IsAuthRestart(err) {
		sent, err = api.SendCode(ctx, phone, auth.SendCodeOptions{})
	}
	if err != nil {
		return fmt.Errorf("send login code: %w", err)
	}
	switch s := sent.(type) {
	case *tg.AuthSentCode:
		if err := describeCodeDelivery(s.Type); err != nil {
			return err
		}
		return signIn(ctx, api, phone, s.PhoneCodeHash)
	case *tg.AuthSentCodeSuccess:
		return nil
	default:
		return fmt.Errorf("unexpected sent code response: %T", sent)
	}
}

func signIn(ctx context.Context, api *auth.Client, phone, hash string) error {
	for attempt := 1; ; attempt++ {
		code, err := promptio.ReadLine(ctx, "Enter the code: ")
		if err != nil {
			return err
		}

		_, err = api.SignIn(ctx, phone, code, hash)
		switch {
		case err == nil:
			return nil
		case errors.Is(err, auth.ErrPasswordAuthNeeded):
			return password(ctx, api)
		case tg.IsPhoneCodeInvalid(err) && attempt < maxAuthAttempts:
			fmt.Println("Invalid code, try again.")
		default:
			return err
		}
	}
}

func password(ctx context.Context, api *auth.Client) error {
	for attempt := 1; ; attempt++ {
		pwd, err := promptio.ReadPassword(ctx, "Two-step verification is enabled. Enter your password: ")
		if err != nil {
			return err
		}

		_, err = api.Password(ctx, pwd)
		switch {
		case err == nil:
			return nil
		case errors.Is(err, auth.ErrPasswordInvalid) && attempt < maxAuthAttempts:
			fmt.Println("Invalid password, try again.")
		default:
			return fmt.Errorf("sign in with 2FA password: %w", err)
		}
	}
}

func describeCodeDelivery(t tg.AuthSentCodeTypeClass) error {
	switch t := t.(type) {
	case *tg.AuthSentCodeTypeApp:
		fmt.Println(`Code sent to your Telegram app: check the "Telegram" service chat on another logged-in device (not SMS).`)
	case *tg.AuthSentCodeTypeSMS, *tg.AuthSentCodeTypeSMSWord, *tg.AuthSentCodeTypeSMSPhrase, *tg.AuthSentCodeTypeFirebaseSMS:
		fmt.Println("Code sent via SMS.")
	case *tg.AuthSentCodeTypeCall:
		fmt.Println("You will receive a phone call with the code.")
	case *tg.AuthSentCodeTypeFlashCall, *tg.AuthSentCodeTypeMissedCall:
		fmt.Println("You will receive a call; the code is the last digits of the calling number.")
	case *tg.AuthSentCodeTypeEmailCode:
		fmt.Printf("Code sent to your login email (%s).\n", t.EmailPattern)
	case *tg.AuthSentCodeTypeFragmentSMS:
		fmt.Printf("Code sent to Fragment: %s\n", t.URL)
	case *tg.AuthSentCodeTypeSetUpEmailRequired:
		return errors.New("telegram requires a login email for this account: set one up in the official Telegram app (Settings > Privacy and Security), then retry")
	default:
		fmt.Printf("Code sent (%T).\n", t)
	}
	return nil
}

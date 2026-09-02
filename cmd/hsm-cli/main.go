package main

import (
	"context"
	"crypto/sha256"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/sergiorandria/hsm-go-client/hsm"
	_ "github.com/sergiorandria/hsm-go-client/hsm/cloudhsm"
	_ "github.com/sergiorandria/hsm-go-client/hsm/luna"
	_ "github.com/sergiorandria/hsm-go-client/hsm/yubihsm"
)

func main() {
	var (
		backend   = flag.String("backend", "pkcs11", "Backend: http, pkcs11, yubihsm, luna, cloudhsm")
		lib       = flag.String("lib", "", "PKCS#11 library path")
		token     = flag.String("token", "", "Token label")
		pin       = flag.String("pin", "", "PIN (or HSM_PIN env)")
		label     = flag.String("label", "mykey", "Key label")
		action    = flag.String("action", "info", "Action: info, generate, list, sign, health")
		message   = flag.String("message", "hello", "Message to sign (for sign)")
		httpURL   = flag.String("http-url", "http://192.168.0.102", "HTTP BaseURL for http backend")
	)
	flag.Parse()

	if *pin == "" {
		*pin = string(hsm.PINFromEnv())
	}
	if *pin == "" {
		*pin = os.Getenv("HSM_PIN")
	}

	cfg := hsm.DriverConfig{Backend: *backend}
	switch *backend {
	case "http", "mcu", "esp32":
		cfg.HTTP = hsm.HTTPConfig{BaseURL: *httpURL}
		if *lib != "" {
			// lib not used for http
		}
	default:
		if *lib == "" {
			// Default to SoftHSM for demo
			*lib = "/usr/lib/softhsm/libsofthsm2.so"
		}
		if *token == "" {
			*token = "test-token"
		}
		cfg.PKCS11 = hsm.PKCS11Config{LibraryPath: *lib, TokenLabel: *token, PIN: *pin}
	}

	driver, err := hsm.NewDriver(cfg, hsm.WithLogger(nil))
	if err != nil {
		log.Fatalf("NewDriver: %v", err)
	}
	defer driver.Close()

	ctx := context.Background()
	switch *action {
	case "info", "health":
		info, err := driver.Info(ctx)
		if err != nil {
			log.Fatalf("Info: %v", err)
		}
		fmt.Printf("Backend %s Slot %d Token %s Manufacturer %s\n", *backend, info.SlotID, info.TokenLabel, info.ManufacturerID)
		health, _ := hsm.Health(ctx, driver, *backend)
		fmt.Printf("Health: %s latency %v\n", health.Status, health.Latency)
	case "generate":
		ki, err := driver.GenerateKey(ctx, hsm.KeySpec{Label: *label, Curve: "P-256"})
		if err != nil {
			log.Fatalf("GenerateKey: %v", err)
		}
		fmt.Printf("Generated %s %s\n", ki.Algorithm, ki.ID.Label)
		fmt.Println(ki.PublicKeyPEM)
	case "list":
		keys, err := driver.ListKeys(ctx)
		if err != nil {
			log.Fatalf("ListKeys: %v", err)
		}
		for _, k := range keys {
			fmt.Printf("%s %s\n", k.ID.Label, k.Algorithm)
		}
	case "sign":
		digest := sha256.Sum256([]byte(*message))
		sig, err := driver.Sign(ctx, hsm.KeyID{Label: *label}, digest[:], hsm.MechanismECDSASHA256)
		if err != nil {
			log.Fatalf("Sign: %v", err)
		}
		fmt.Printf("Signature %x (%d bytes)\n", sig, len(sig))
	default:
		log.Fatalf("unknown action %q", *action)
	}
}

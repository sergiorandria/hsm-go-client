// Package luna provides a Thales Luna Network HSM adapter for hsm.Driver.
//
// Luna uses Chrystoki PKCS#11 (libCryptoki2.so / libLunaAPI.so) with partition-based
// slots and HA. Generic hsm/pkcs11 assumes simple PIN + CKM_ECDSA with raw digest.
// Contradictions handled without modifying generic driver:
//
//   - Library: /usr/safenet/lunaclient/lib/libCryptoki2_64.so (also libLunaAPI.so)
//   - Slot: partition label via TokenLabel or SlotID; Luna HA uses virtual slot
//   - PIN: partition challenge (partition password), not SO PIN; may need Protected Authentication flag
//   - Mechanism: Luna supports CKM_ECDSA_SHA256 etc. Generic uses CKM_ECDSA for raw digest.
//     Adapter keeps raw CKM_ECDSA (Luna docs allow it) but documents alternative.
//   - RSA-PSS: requires CK_RSA_PKCS_PSS_PARAMS (hashAlg, mgf, saltLen) — generic passes nil.
//     Adapter notes this and exposes PSS config for future override.
//   - Chrystoki.conf + HA config via /etc/Chrystoki.conf — adapter checks SlotDescription contains "Luna"
//
// Usage:
//
//	import "github.com/sergiorandria/hsm-go-client/hsm/luna"
//	d, err := luna.NewDriver(luna.Config{
//	  LibraryPath: "/usr/safenet/lunaclient/lib/libCryptoki2_64.so",
//	  TokenLabel:  "myPartition",
//	  PIN:         "partitionPassword",
//	})
package luna

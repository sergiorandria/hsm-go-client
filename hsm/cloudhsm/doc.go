// Package cloudhsm provides an AWS CloudHSM adapter for hsm.Driver.
//
// CloudHSM (Cavium/Marvell, v5) is PKCS#11 via libcloudhsm_pkcs11.so with cluster daemon.
// Generic hsm/pkcs11 assumes single slot + PIN. Contradictions handled without modifying generic:
//
//   - Library: /opt/cloudhsm/lib/libcloudhsm_pkcs11.so
//   - Slot: cluster token label, CU user via PIN "username:password" (e.g. "alice:StrongPass123")
//   - HA: cluster sync async — GenerateKey may need ListKeys retry; session pool handles CKR_DEVICE_REMOVED
//   - Daemon: cloudhsm_client + cloudhsm_mgmt_util must configure cluster before use
//   - Mechanism: same as generic (CKM_ECDSA for raw digest), but CloudHSM enforces CKA_EXTRACTABLE=false
//
// Adapter wraps generic driver and normalizes CU PIN format and cluster retry.
//
// Usage:
//
//	import "github.com/sergiorandria/hsm-go-client/hsm/cloudhsm"
//	d, err := cloudhsm.NewDriver(cloudhsm.Config{
//	  LibraryPath: "/opt/cloudhsm/lib/libcloudhsm_pkcs11.so",
//	  TokenLabel:  "cavium",
//	  PIN:         "alice:StrongPass123",
//	})
package cloudhsm

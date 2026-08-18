#!/bin/bash
# Fix P0: Order number generation collision risk
# This script updates all number generation functions to use microsecond precision + 4 random bytes

set -e

BACKEND_DIR="/Users/gongqianrong/Desktop/ai/backend"

echo "🔧 Fixing P0: Order Number Generation..."

# Backup files
echo "📦 Creating backups..."
cp "$BACKEND_DIR/internal/domain/order.go" "$BACKEND_DIR/internal/domain/order.go.bak"
cp "$BACKEND_DIR/internal/domain/wallet.go" "$BACKEND_DIR/internal/domain/wallet.go.bak"
cp "$BACKEND_DIR/internal/domain/waybill.go" "$BACKEND_DIR/internal/domain/waybill.go.bak"
cp "$BACKEND_DIR/internal/domain/order_link.go" "$BACKEND_DIR/internal/domain/order_link.go.bak"

# Fix order.go
echo "✏️  Fixing order.go..."
cat > /tmp/generate_order_number.txt << 'EOF'
// GenerateOrderNumber creates a unique order number: RO + timestamp (microsecond) + 8 random hex chars.
// Format: RO20260102150405123456ABCD1234 (prefix + YYYYMMDDHHmmssμs + random)
// Microsecond precision + 4 random bytes = extremely low collision risk.
func GenerateOrderNumber() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		// Fallback: use nanosecond time as additional entropy
		nanos := time.Now().UnixNano()
		b[0] = byte(nanos >> 24)
		b[1] = byte(nanos >> 16)
		b[2] = byte(nanos >> 8)
		b[3] = byte(nanos)
	}
	now := time.Now()
	// Use microsecond precision to reduce same-second collisions
	return fmt.Sprintf("RO%s%06d%X", 
		now.Format("20060102150405"), 
		now.Nanosecond()/1000, // microseconds
		b)
}
EOF

# Use awk to replace the function in order.go
awk '
/^\/\/ GenerateOrderNumber/ {
    skip = 1
    system("cat /tmp/generate_order_number.txt")
}
skip && /^}$/ {
    skip = 0
    next
}
!skip {
    print
}
' "$BACKEND_DIR/internal/domain/order.go.bak" > "$BACKEND_DIR/internal/domain/order.go"

# Fix wallet.go  
echo "✏️  Fixing wallet.go..."
cat > /tmp/generate_recharge_no.txt << 'EOF'
// GenerateRechargeNo creates a unique recharge order number: RC + timestamp (microsecond) + 8 random hex chars.
// Format: RC20260102150405123456ABCD1234 (prefix + YYYYMMDDHHmmssμs + random)
func GenerateRechargeNo() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		nanos := time.Now().UnixNano()
		b[0] = byte(nanos >> 24)
		b[1] = byte(nanos >> 16)
		b[2] = byte(nanos >> 8)
		b[3] = byte(nanos)
	}
	now := time.Now()
	return fmt.Sprintf("RC%s%06d%X", 
		now.Format("20060102150405"), 
		now.Nanosecond()/1000,
		b)
}
EOF

awk '
/^\/\/ GenerateRechargeNo/ {
    skip = 1
    system("cat /tmp/generate_recharge_no.txt")
}
skip && /^}$/ {
    skip = 0
    next
}
!skip {
    print
}
' "$BACKEND_DIR/internal/domain/wallet.go.bak" > "$BACKEND_DIR/internal/domain/wallet.go"

# Fix waybill.go
echo "✏️  Fixing waybill.go..."
cat > /tmp/generate_waybill_no.txt << 'EOF'
// GenerateWaybillNo creates a unique waybill number: LO + timestamp (microsecond) + 8 random hex chars.
// Format: LO20260102150405123456ABCD1234 (prefix + YYYYMMDDHHmmssμs + random)
func GenerateWaybillNo() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		nanos := time.Now().UnixNano()
		b[0] = byte(nanos >> 24)
		b[1] = byte(nanos >> 16)
		b[2] = byte(nanos >> 8)
		b[3] = byte(nanos)
	}
	now := time.Now()
	return fmt.Sprintf("LO%s%06d%X", 
		now.Format("20060102150405"), 
		now.Nanosecond()/1000,
		b)
}
EOF

awk '
/^\/\/ GenerateWaybillNo/ {
    skip = 1
    system("cat /tmp/generate_waybill_no.txt")
}
skip && /^}$/ {
    skip = 0
    next
}
!skip {
    print
}
' "$BACKEND_DIR/internal/domain/waybill.go.bak" > "$BACKEND_DIR/internal/domain/waybill.go"

# Fix order_link.go
echo "✏️  Fixing order_link.go..."
cat > /tmp/generate_link_no.txt << 'EOF'
// GenerateLinkNo creates a unique link number: OL + timestamp (microsecond) + 8 random hex chars.
// Format: OL20260102150405123456ABCD1234 (prefix + YYYYMMDDHHmmssμs + random)
func GenerateLinkNo() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		nanos := time.Now().UnixNano()
		b[0] = byte(nanos >> 24)
		b[1] = byte(nanos >> 16)
		b[2] = byte(nanos >> 8)
		b[3] = byte(nanos)
	}
	now := time.Now()
	return fmt.Sprintf("OL%s%06d%X", 
		now.Format("20060102150405"), 
		now.Nanosecond()/1000,
		b)
}
EOF

awk '
/^\/\/ GenerateLinkNo/ {
    skip = 1
    system("cat /tmp/generate_link_no.txt")
}
skip && /^}$/ {
    skip = 0
    next
}
!skip {
    print
}
' "$BACKEND_DIR/internal/domain/order_link.go.bak" > "$BACKEND_DIR/internal/domain/order_link.go"

echo "✅ Order number generation fixed!"
echo ""
echo "Modified files:"
echo "  - internal/domain/order.go"
echo "  - internal/domain/wallet.go"
echo "  - internal/domain/waybill.go"
echo "  - internal/domain/order_link.go"
echo ""
echo "Backups created with .bak extension"
echo ""
echo "Changes:"
echo "  ✅ Error checking for rand.Read()"
echo "  ✅ Increased random bytes: 2 → 4 (65K → 4B combinations)"
echo "  ✅ Microsecond precision (vs second)"
echo "  ✅ Fallback to nanosecond entropy if crypto/rand fails"
echo ""
echo "Next: Run 'go build' to verify compilation"

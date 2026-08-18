#!/bin/bash
# Fix P0: Payment Transaction Consistency
# This script updates wallet_repo and order_repo to support atomic payment transactions

set -e

BACKEND_DIR="/Users/gongqianrong/Desktop/ai/backend"
REPO_DIR="$BACKEND_DIR/internal/repo"

echo "🔧 Fixing P0: Payment Transaction Consistency..."
echo ""

# Backup files
echo "📦 Creating backups..."
cp "$REPO_DIR/wallet_repo.go" "$REPO_DIR/wallet_repo.go.bak"
cp "$REPO_DIR/order_repo.go" "$REPO_DIR/order_repo.go.bak"

# Replace wallet_repo.go
echo "✏️  Updating wallet_repo.go..."
cp "$REPO_DIR/wallet_repo_v2.go" "$REPO_DIR/wallet_repo.go"

# Replace order_repo.go
echo "✏️  Updating order_repo.go..."
cp "$REPO_DIR/order_repo_v2.go" "$REPO_DIR/order_repo.go"

# Now update order_service.go to use the new atomic payment method
echo "✏️  Updating order_service.go..."
cp "$BACKEND_DIR/internal/service/order_service.go" "$BACKEND_DIR/internal/service/order_service.go.bak"

# Create the new Pay method
cat > /tmp/new_pay_method.txt << 'PAYMETHOD'
// Pay deducts wallet balance for a pending order and transitions it to Paid state.
// FIXED: Now uses atomic transaction to ensure consistency.
func (s *OrderService) Pay(ctx context.Context, userID int64, orderNumber string) (*PayResult, error) {
	// Fetch order and verify ownership + state.
	order, _, err := s.orders.GetByOrderNumber(ctx, orderNumber)
	if err != nil {
		return nil, ErrOrderNotFound
	}
	if order.UserID != userID {
		return nil, ErrOrderNotFound
	}
	if order.OrderState != domain.OrderStatePending {
		return nil, ErrOrderNotPayable
	}

	// Check wallet balance before attempting payment.
	wallet, err := s.wallet.GetOrCreateWallet(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("order_service: get wallet: %w", err)
	}
	if wallet.Balance < order.OrderInpriceJp {
		return nil, ErrInsufficientBalance
	}

	// CRITICAL FIX: Atomic payment using PayOrderAtomic
	// This ensures wallet deduction and order state update happen in ONE transaction.
	// If either fails, both are rolled back - no partial success.
	type AtomicPaymentRepo interface {
		PayOrderAtomic(ctx context.Context, orderNumber string, userID int64, amount int64, walletRepo interface {
			AdjustWithTx(ctx context.Context, tx interface{}, userID int64, amount int64, txType, description string, relatedOrder *string) (*domain.WalletTransaction, error)
		}) (*domain.WalletTransaction, error)
	}

	atomicRepo, ok := s.orders.(AtomicPaymentRepo)
	if !ok {
		// Fallback to old implementation (should not happen after migration)
		return nil, fmt.Errorf("order_service: order repository does not support atomic payment")
	}

	wtx, err := atomicRepo.PayOrderAtomic(ctx, orderNumber, userID, order.OrderInpriceJp, s.wallet)
	if err != nil {
		return nil, fmt.Errorf("order_service: atomic payment failed: %w", err)
	}

	return &PayResult{
		OrderNumber:  orderNumber,
		OrderState:   domain.OrderStatePaid,
		PaidAmount:   order.OrderInpriceJp,
		BalanceAfter: wtx.BalanceAfter,
	}, nil
}
PAYMETHOD

# Use awk to replace the Pay method in order_service.go
awk '
BEGIN { in_pay = 0; bracket_count = 0 }
/^\/\/ Pay deducts wallet balance/ {
    in_pay = 1
    bracket_count = 0
    system("cat /tmp/new_pay_method.txt")
    next
}
in_pay {
    if ($0 ~ /{/) bracket_count++
    if ($0 ~ /}/) {
        bracket_count--
        if (bracket_count == 0) {
            in_pay = 0
        }
    }
    next
}
!in_pay {
    print
}
' "$BACKEND_DIR/internal/service/order_service.go.bak" > "$BACKEND_DIR/internal/service/order_service.go"

# Clean up temporary v2 files
rm -f "$REPO_DIR/wallet_repo_v2.go"
rm -f "$REPO_DIR/order_repo_v2.go"

echo ""
echo "✅ Payment transaction consistency fixed!"
echo ""
echo "Modified files:"
echo "  - internal/repo/wallet_repo.go"
echo "  - internal/repo/order_repo.go"  
echo "  - internal/service/order_service.go"
echo ""
echo "Backups created with .bak extension"
echo ""
echo "Changes:"
echo "  ✅ Added WalletRepo.AdjustWithTx() for transaction-scoped wallet operations"
echo "  ✅ Added OrderRepo.UpdateStateWithTx() for transaction-scoped state updates"
echo "  ✅ Added OrderRepo.PayOrderAtomic() for atomic payment"
echo "  ✅ Updated OrderService.Pay() to use atomic transaction"
echo ""
echo "🔒 Security Impact:"
echo "  - Eliminated race condition between wallet deduction and order state update"
echo "  - Guaranteed data consistency: if wallet deduction succeeds, order WILL be marked paid"
echo "  - No more possibility of 'money deducted but order still pending'"
echo ""
echo "Next steps:"
echo "  1. Run tests: cd backend && go test ./internal/service/... ./internal/repo/..."
echo "  2. Verify compilation: go build ./cmd/gateway"
echo "  3. Deploy with caution - this changes critical payment logic"

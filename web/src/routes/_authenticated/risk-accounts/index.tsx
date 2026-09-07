import { createFileRoute, redirect } from '@tanstack/react-router'
import { RiskAccounts } from '@/features/risk-accounts'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

export const Route = createFileRoute('/_authenticated/risk-accounts/')({
  beforeLoad: () => { if ((useAuthStore.getState().auth.user?.role ?? 0) < ROLE.ADMIN) throw redirect({ to: '/dashboard' }) },
  component: RiskAccounts,
})

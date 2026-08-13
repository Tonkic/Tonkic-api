/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/
import { createFileRoute } from '@tanstack/react-router'

import { CompensationClaimPage } from '@/features/compensation-campaigns/claim-page'

export const Route = createFileRoute('/_authenticated/claim/$code')({
  component: ClaimRoute,
})

function ClaimRoute() {
  const { code } = Route.useParams()
  return <CompensationClaimPage code={code} />
}

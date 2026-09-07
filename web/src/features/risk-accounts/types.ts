export type RiskCaseUser = {
  case_id: number
  user_id: number
  username: string
  score: number
  evidence_json: string
}

export type RiskCase = {
  id: number
  status: 'open' | 'ignored' | 'banned'
  score: number
  categories: number
  reason_summary: string
  first_seen: number
  last_seen: number
  users: RiskCaseUser[]
}

export type RiskCasePage = { items: RiskCase[]; total: number }

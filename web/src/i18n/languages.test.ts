/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { describe, expect, it } from 'vitest'

import { toIntlLocale } from './languages'

describe('toIntlLocale', () => {
  it.each([
    ['zhCN', 'zh-CN'],
    ['zhTW', 'zh-TW'],
    ['fr', 'fr'],
  ])('maps interface language %s to Intl locale %s', (input, expected) => {
    expect(toIntlLocale(input)).toBe(expected)
    expect(() => new Intl.DateTimeFormat(toIntlLocale(input))).not.toThrow()
  })

  it('falls back to the runtime locale for invalid language tags', () => {
    expect(toIntlLocale('not_a_locale')).toBeUndefined()
    expect(() => new Intl.DateTimeFormat(toIntlLocale('not_a_locale'))).not.toThrow()
  })
})

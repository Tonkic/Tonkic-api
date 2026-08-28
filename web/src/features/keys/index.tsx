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
import { AlertTriangle } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { CopyButton } from '@/components/copy-button'

import { ApiKeysDialogs } from './components/api-keys-dialogs'
import { ApiKeysPrimaryButtons } from './components/api-keys-primary-buttons'
import { ApiKeysProvider } from './components/api-keys-provider'
import { ApiKeysTable } from './components/api-keys-table'

export function ApiKeys() {
  const { t } = useTranslation()
  return (
    <ApiKeysProvider>
      <SectionPageLayout fixedContent>
        <SectionPageLayout.Title>{t('API Keys')}</SectionPageLayout.Title>
        <SectionPageLayout.Actions>
          <ApiKeysPrimaryButtons />
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          <div className='mb-4 flex items-start gap-3 rounded-md border border-amber-500/40 bg-amber-500/10 p-3 text-sm text-amber-900 dark:text-amber-100'>
            <AlertTriangle className='mt-0.5 size-4 shrink-0' aria-hidden='true' />
            <div className='flex flex-wrap items-center gap-x-3 gap-y-2'>
              <span>{t('TonkicAPI service notice')}</span>
              <span className='inline-flex items-center gap-1.5'>
                {t('China site')}:
                <a className='font-mono underline' href='https://tonkicapi.xyz' target='_blank' rel='noreferrer'>https://tonkicapi.xyz</a>
                <CopyButton value='https://tonkicapi.xyz' size='icon-xs' tooltip={t('Copy URL')} aria-label={t('Copy China site URL')} />
              </span>
              <span className='inline-flex items-center gap-1.5'>
                {t('US site')}:
                <a className='font-mono underline' href='https://us.tonkicapi.xyz' target='_blank' rel='noreferrer'>https://us.tonkicapi.xyz</a>
                <CopyButton value='https://us.tonkicapi.xyz' size='icon-xs' tooltip={t('Copy URL')} aria-label={t('Copy US site URL')} />
              </span>
            </div>
          </div>
          <ApiKeysTable />
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <ApiKeysDialogs />
    </ApiKeysProvider>
  )
}

import { Construction } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { ConfigDrawer } from '@/components/config-drawer'
import { LanguageSwitch } from '@/components/language-switch'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { ProfileDropdown } from '@/components/profile-dropdown'
import { Search } from '@/components/search'
import { ThemeSwitch } from '@/components/theme-switch'

/**
 * Shared "under construction" page for routes whose feature has not shipped
 * yet: standard header plus a centered placeholder body.
 */
export function PlaceholderPage() {
  const { t } = useTranslation()

  return (
    <>
      <Header>
        <Search className='me-auto' />
        <ThemeSwitch />
        <LanguageSwitch />
        <ConfigDrawer />
        <ProfileDropdown />
      </Header>
      <Main className='flex h-[calc(100svh-(var(--spacing)*16))]'>
        <div className='m-auto flex w-full flex-col items-center justify-center gap-2'>
          <Construction size={72} />
          <h1 className='text-4xl leading-tight font-bold'>
            {t('placeholder.title')}
          </h1>
          <p className='text-center text-muted-foreground'>
            {t('placeholder.description')}
          </p>
        </div>
      </Main>
    </>
  )
}

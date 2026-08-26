import { Construction } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { ConfigDrawer } from '@/components/config-drawer'
import { LanguageSwitch } from '@/components/language-switch'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { ProfileDropdown } from '@/components/profile-dropdown'
import { Search } from '@/components/search'
import { ThemeSwitch } from '@/components/theme-switch'

type PlaceholderPageProps = {
  /** i18n key for the heading; defaults to the generic placeholder copy. */
  titleKey?: string
  /** i18n key for the description; defaults to the generic copy. */
  descriptionKey?: string
}

/**
 * Shared "under construction" page for routes whose feature has not
 * shipped yet: standard header plus a centered placeholder body.
 * Callers may pass domain-specific i18n keys (e.g. `config.plans.title`)
 * so the page announces its domain until the real feature ships.
 */
export function PlaceholderPage({
  titleKey = 'placeholder.title',
  descriptionKey = 'placeholder.description',
}: PlaceholderPageProps = {}) {
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
          <h1 className='text-4xl leading-tight font-bold'>{t(titleKey)}</h1>
          <p className='text-center text-muted-foreground'>
            {t(descriptionKey)}
          </p>
        </div>
      </Main>
    </>
  )
}

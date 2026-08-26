import { type LinkProps } from '@tanstack/react-router'

type BaseNavItem = {
  /** i18n key resolved at render time. */
  title: string
  /** i18n key for the badge, when present. */
  badge?: string
  icon?: React.ElementType
}

type NavLink = BaseNavItem & {
  url: LinkProps['to'] | (string & {})
  items?: never
}

type NavCollapsible = BaseNavItem & {
  items: (BaseNavItem & { url: LinkProps['to'] | (string & {}) })[]
  url?: never
}

/** Entry without a route yet; rendered inert (badge hints at its status). */
type NavPlaceholder = BaseNavItem & {
  disabled: true
  url?: never
  items?: never
}

type NavItem = NavCollapsible | NavLink | NavPlaceholder

type NavGroup = {
  /** i18n key for the group label. */
  title: string
  items: NavItem[]
}

type SidebarData = {
  navGroups: NavGroup[]
}

export type {
  SidebarData,
  NavGroup,
  NavItem,
  NavCollapsible,
  NavLink,
  NavPlaceholder,
}

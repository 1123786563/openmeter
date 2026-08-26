import {
  Bell,
  Blocks,
  Coins,
  FileText,
  Gauge,
  KeyRound,
  LayoutDashboard,
  ListChecks,
  PackageOpen,
  Plug,
  Puzzle,
  ReceiptText,
  RefreshCw,
  ShoppingCart,
  Undo2,
  Users,
  Zap,
} from 'lucide-react'
import { type SidebarData } from '../types'

// Titles are i18n keys (resolved via t()); badges follow the same rule.
export const sidebarData: SidebarData = {
  navGroups: [
    {
      title: 'sidebar.groups.overview',
      items: [
        {
          title: 'sidebar.dashboard',
          url: '/',
          icon: LayoutDashboard,
        },
      ],
    },
    {
      title: 'sidebar.groups.billing',
      items: [
        {
          title: 'sidebar.customers',
          url: '/customers',
          icon: Users,
        },
        {
          title: 'sidebar.subscriptions',
          url: '/subscriptions',
          icon: RefreshCw,
        },
        {
          title: 'sidebar.invoices',
          url: '/invoices',
          icon: FileText,
        },
      ],
    },
    {
      title: 'sidebar.groups.metering',
      items: [
        {
          title: 'sidebar.meters',
          url: '/meters',
          icon: Gauge,
        },
        {
          title: 'sidebar.events',
          url: '/events',
          icon: Zap,
        },
      ],
    },
    {
      title: 'sidebar.groups.credits',
      items: [
        {
          title: 'sidebar.credits',
          url: '/credits',
          icon: Coins,
        },
      ],
    },
    {
      // Commerce domain (fork-specific): recharge product catalog with
      // admin writes, plus paginated order/refund lists with detail views.
      title: 'sidebar.groups.commerce',
      items: [
        {
          title: 'sidebar.rechargeProducts',
          url: '/commerce/recharge-products',
          icon: PackageOpen,
        },
        {
          title: 'sidebar.orders',
          url: '/commerce/orders',
          icon: ShoppingCart,
        },
        {
          title: 'sidebar.refunds',
          url: '/commerce/refunds',
          icon: Undo2,
        },
      ],
    },
    {
      // Configuration domain (issues #1-#29): plans/features/add-ons,
      // notification center, currencies & tax codes, apps/portal tokens/
      // billing profiles. Placeholder routes for now; issues #2-#28
      // replace the components with the real pages.
      title: 'sidebar.groups.config',
      items: [
        {
          title: 'sidebar.plans',
          url: '/config/plans',
          icon: ListChecks,
        },
        {
          title: 'sidebar.features',
          url: '/config/features',
          icon: Blocks,
        },
        {
          title: 'sidebar.addons',
          url: '/config/addons',
          icon: Puzzle,
        },
        {
          title: 'sidebar.notificationChannels',
          url: '/config/notification/channels',
          icon: Bell,
        },
        {
          title: 'sidebar.currencies',
          url: '/config/currencies',
          icon: Coins,
        },
        {
          title: 'sidebar.taxCodes',
          url: '/config/tax-codes',
          icon: ReceiptText,
        },
        {
          title: 'sidebar.apps',
          url: '/config/apps',
          icon: Plug,
        },
        {
          title: 'sidebar.portalTokens',
          url: '/config/portal-tokens',
          icon: KeyRound,
        },
        {
          title: 'sidebar.billingProfiles',
          url: '/config/billing-profiles',
          icon: FileText,
        },
      ],
    },
  ],
}

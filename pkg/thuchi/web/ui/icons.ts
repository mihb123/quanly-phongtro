import type { ComponentType } from 'react'
import {
  Bank, Bill, Book, Car, Cloud, Fuel, Game, House, Key, Medicine, MoneyIn, MoreDots, Perfume, Restaurant, SendMoney, ShoppingBag, Sparkles,
  Tag, Utensils, Wallet, Money, CreditCard, Cash, Coffee, Wrench, Zap, Droplet, Droplets, Tools, PaintBrush, Sofa, Fridge, PackageBox, Broom, Waste,
  Toilet, PlugSocket, Wifi, Handshake, Megaphone, Taxes, FireExtinguisher, Users, Undo, Percent, Coins, PiggyBank, type IconProps,
} from '@/components/icons'
import { isDebtKind, type Category, type Transaction } from '../types'

export const ICONS: Record<string, ComponentType<IconProps>> = {
  food: Utensils,
  restaurant: Restaurant,
  shopping: ShoppingBag,
  cosmetic: Perfume,
  beauty: Sparkles,
  car: Car,
  fuel: Fuel,
  medicine: Medicine,
  book: Book,
  cloud: Cloud,
  game: Game,
  bill: Bill,
  home: House,
  send: SendMoney,
  bank: Bank,
  more: MoreDots,
  salary: Money,
  key: Key,
  receive: MoneyIn,
  coffee: Coffee,
  tag: Tag,
  wallet: Wallet,
  card: CreditCard,
  cash: Cash,
  repair: Wrench,
  electric: Zap,
  water: Droplet,
  tools: Tools,
  paint: PaintBrush,
  sofa: Sofa,
  fridge: Fridge,
  package: PackageBox,
  clean: Broom,
  waste: Waste,
  toilet: Toilet,
  plug: PlugSocket,
  droplets: Droplets,
  wifi: Wifi,
  handshake: Handshake,
  megaphone: Megaphone,
  tax: Taxes,
  fire: FireExtinguisher,
  worker: Users,
  undo: Undo,
  percent: Percent,
  coins: Coins,
  piggy: PiggyBank,
}

export const ICON_KEYS = Object.keys(ICONS)

export const COLORS = ['#f97316', '#ef4444', '#ec4899', '#d946ef', '#a855f7', '#6366f1', '#3b82f6', '#0ea5e9', '#14b8a6', '#22c55e', '#84cc16', '#eab308', '#78716c', '#64748b', '#94a3b8']

export const KIND_LABEL: Partial<Record<Transaction['kind'], string>> = {
  transfer: 'Chuyển nội bộ',
  refund: 'Hoàn tiền',
  card_payment: 'Trả thẻ',
  adjustment: 'Điều chỉnh số dư',
  borrow: 'Đi vay',
  lend: 'Cho vay',
  repay: 'Trả nợ',
  collect: 'Thu nợ',
}

export function txTitle(tx: Transaction, category?: Category | null) {
  if (tx.kind === 'transfer') return tx.merchant || tx.note || 'Chuyển nội bộ'
  if (tx.kind === 'adjustment') return tx.note || 'Điều chỉnh số dư'
  if (isDebtKind(tx.kind)) return tx.merchant || [KIND_LABEL[tx.kind], tx.party].filter(Boolean).join(' · ')
  return tx.merchant || tx.counterparty_name || tx.note || category?.name || tx.raw_content || 'Giao dịch'
}


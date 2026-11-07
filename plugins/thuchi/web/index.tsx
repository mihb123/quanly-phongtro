import { Wallet } from '@/components/icons'
import { definePlugin } from '@/plugins/sdk'
import { ThuChiView } from './ThuChiView'

export default definePlugin({
  id: 'thu-chi',
  icon: Wallet,
  View: ThuChiView,
})

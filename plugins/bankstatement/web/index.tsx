import { FileUp } from '@/components/icons'
import { definePlugin } from '@/plugins/sdk'
import { BankStatementView } from './BankStatementView'

export default definePlugin({
  id: 'bank-statement',
  icon: FileUp,
  View: BankStatementView,
})

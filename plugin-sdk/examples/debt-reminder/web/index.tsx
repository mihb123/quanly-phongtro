import { definePlugin } from '@qlpt/plugin'
import { Receipt } from '@qlpt/plugin/icons'
import { DebtReminderView } from './DebtReminderView'
import './style.css'

export default definePlugin({
  id: 'debt-reminder',
  icon: Receipt,
  View: DebtReminderView,
})

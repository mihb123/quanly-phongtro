import { PayOSSettingsCard } from '@/components/home/settings/PayOSSettingsCard'
import { ZaloBotSettingsCard } from '@/components/home/settings/ZaloBotSettingsCard'

export function SettingsView() {
  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-2">
        <h1 className="text-3xl font-bold tracking-tight text-slate-800">Cài đặt</h1>
        <p className="text-muted-foreground">Quản lý các cấu hình tích hợp hệ thống</p>
      </div>

      <ZaloBotSettingsCard />
      <PayOSSettingsCard />
    </div>
  )
}

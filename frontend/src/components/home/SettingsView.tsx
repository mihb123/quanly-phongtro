import { PageHeader } from '@/components/shared/PageHeader'
import { PayOSSettingsCard } from '@/components/home/settings/PayOSSettingsCard'
import { SePaySettingsCard } from '@/components/home/settings/SePaySettingsCard'
import { ZaloBotSettingsCard } from '@/components/home/settings/ZaloBotSettingsCard'

// View tổng hợp các card cấu hình tích hợp hệ thống (Zalo, SePay, PayOS).
export function SettingsView() {
  return (
    <div className="space-y-6">
      <PageHeader title="Cài đặt" description="Quản lý các cấu hình tích hợp hệ thống" />

      <ZaloBotSettingsCard />
      <SePaySettingsCard />
      <PayOSSettingsCard />
    </div>
  )
}

import { useState, useEffect } from 'react'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { getZaloPublicKey, getZaloConfigStatus, saveZaloConfig } from '@/api/zalo'
import { toast } from 'sonner'
import { Loader2, CheckCircle2, AlertCircle, Eye, EyeOff } from 'lucide-react'
import { useAuth } from '@/contexts/AuthContext'

const encryptRSA = async (text: string, spkiBase64: string) => {
  const binaryString = window.atob(spkiBase64);
  const bytes = new Uint8Array(binaryString.length);
  for (let i = 0; i < binaryString.length; i++) {
    bytes[i] = binaryString.charCodeAt(i);
  }

  const cryptoKey = await window.crypto.subtle.importKey(
    "spki",
    bytes,
    {
      name: "RSA-OAEP",
      hash: "SHA-256",
    },
    true,
    ["encrypt"]
  );

  const enc = new TextEncoder();
  const encryptedBuf = await window.crypto.subtle.encrypt(
    { name: "RSA-OAEP" },
    cryptoKey,
    enc.encode(text)
  );

  return window.btoa(String.fromCharCode(...new Uint8Array(encryptedBuf)));
};

export function SettingsView() {
  const [botToken, setBotToken] = useState('')
  const [showBotToken, setShowBotToken] = useState(false)
  const [hasConfig, setHasConfig] = useState<boolean | null>(null)
  const [isLinked, setIsLinked] = useState<boolean>(false)
  const [botId, setBotId] = useState<string>('')
  const [managerId, setManagerId] = useState<string>('')
  const [loading, setLoading] = useState(false)
  const [initialLoading, setInitialLoading] = useState(true)
  const { user } = useAuth()

  const isMobile = /iPhone|iPad|iPod|Android/i.test(navigator.userAgent);
  const managerName = user?.full_name || managerId;

  useEffect(() => {
    const fetchStatus = async () => {
      try {
        const res = await getZaloConfigStatus()
        setHasConfig(res.has_config)
        setIsLinked(res.is_linked)
        setBotId(res.bot_id)
        setManagerId(res.manager_id)
      } catch (err) {
        console.error("Failed to load zalo config status", err)
      } finally {
        setInitialLoading(false)
      }
    }
    fetchStatus()
  }, [user?.user_id])

  const handleSave = async () => {
    if (!botToken) {
      toast.error('Vui lòng nhập Bot Token')
      return
    }

    setLoading(true)
    try {
      // Fetch server's public key
      const { public_key } = await getZaloPublicKey()

      // Encrypt payloads
      const encryptedToken = await encryptRSA(botToken, public_key)

      // Send to server
      await saveZaloConfig({
        bot_token: encryptedToken
      })

      toast.success('Cấu hình Zalo Bot thành công')
      setBotToken('')
      
      // Re-fetch status to get the bot_id and check link status
      const res = await getZaloConfigStatus()
      setHasConfig(res.has_config)
      setIsLinked(res.is_linked)
      setBotId(res.bot_id)
      setManagerId(res.manager_id)
    } catch (err: unknown) {
      const error = err as { response?: { data?: { message?: string } }, message?: string };
      toast.error(error.response?.data?.message || error.message || 'Lỗi cấu hình Zalo')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-2">
        <h1 className="text-3xl font-bold tracking-tight text-slate-800">Cài đặt</h1>
        <p className="text-muted-foreground">Quản lý các cấu hình tích hợp hệ thống</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Tích hợp Zalo Bot</CardTitle>
          <CardDescription>
            Cấu hình Bot để hệ thống tự động gửi thông báo hóa đơn qua Zalo.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {initialLoading ? (
            <div className="flex items-center gap-2 text-slate-500">
              <Loader2 className="w-4 h-4 animate-spin" />
              Đang tải trạng thái...
            </div>
          ) : (
            <div className={`p-4 rounded-xl flex items-start gap-3 border ${hasConfig ? 'bg-green-50 border-green-200 text-green-700' : 'bg-amber-50 border-amber-200 text-amber-700'}`}>
              {hasConfig ? <CheckCircle2 className="w-5 h-5 mt-0.5 flex-shrink-0" /> : <AlertCircle className="w-5 h-5 mt-0.5 flex-shrink-0" />}
              <div>
                <p className="font-semibold">{hasConfig ? 'Đã kết nối Zalo Bot' : 'Chưa cấu hình Zalo Bot'}</p>
                <p className="text-sm opacity-90">
                  {hasConfig 
                    ? 'Hệ thống đã sẵn sàng gửi tin nhắn qua Zalo. Bạn có thể cập nhật Token bên dưới nếu cần thiết.' 
                    : 'Vui lòng cung cấp Bot Token để kích hoạt tính năng.'}
                </p>
              </div>
            </div>
          )}

          {!initialLoading && hasConfig && !isLinked && botId && (
            <div className="p-4 rounded-xl flex items-start gap-3 border bg-blue-50 border-blue-200 text-blue-800">
              <AlertCircle className="w-5 h-5 mt-0.5 flex-shrink-0 text-blue-600" />
              <div>
                <p className="font-semibold text-blue-900 mb-3">Yêu cầu hoàn tất liên kết tài khoản</p>                
                {isMobile ? (
                  <a 
                    href={`https://zalo.me/${botId}?text=Kich hoat bot cho tai khoan: ${managerName}`}
                    target="_blank"
                    rel="noreferrer"
                    className="inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50 bg-blue-600 text-white shadow hover:bg-blue-600/90 h-9 px-4 py-2"
                  >
                    Mở ứng dụng Zalo ngay
                  </a>
                ) : (
                  <div className="flex flex-col gap-3 items-start w-full max-w-sm">
                    <a 
                      href={`https://chat.zalo.me/?c=${botId}&text=Kich hoat bot cho tai khoan: ${managerName}`}
                      target="_blank"
                      rel="noreferrer"
                      className="inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50 bg-blue-600 text-white shadow hover:bg-blue-600/90 h-9 px-4 py-2"
                    >
                      Mở Zalo Web / PC
                    </a>
                  </div>
                )}
              </div>
            </div>
          )}

          <div className="grid gap-4 mt-6 max-w-xl">
            <div className="space-y-2">
              <Label htmlFor="bot-token">Bot Token</Label>
              <div className="relative">
                <Input
                  id="bot-token"
                  type={showBotToken ? "text" : "password"}
                  placeholder="Nhập Bot Token"
                  value={botToken}
                  onChange={(e) => setBotToken(e.target.value)}
                  className="pr-10"
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="absolute right-0 top-0 h-full px-3 py-2 hover:bg-transparent"
                  onClick={() => setShowBotToken(!showBotToken)}
                >
                  {showBotToken ? (
                    <EyeOff className="h-4 w-4 text-slate-500" />
                  ) : (
                    <Eye className="h-4 w-4 text-slate-500" />
                  )}
                </Button>
              </div>
            </div>



            <Button onClick={handleSave} disabled={loading} className="w-fit mt-2">
              {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Lưu cấu hình
            </Button>
          </div>

          <div className="mt-8 pt-6 border-t">
            <h3 className="font-semibold text-lg mb-4">Hướng dẫn cấu hình Zalo Bot</h3>
            <div className="space-y-4 text-sm text-slate-700">
              <div className="flex gap-3">
                <div className="w-6 h-6 rounded-full bg-blue-100 text-blue-700 flex items-center justify-center flex-shrink-0 font-bold">1</div>
                <div>
                  <p>Truy cập <a href="https://bot.zapps.me/docs/create-bot" target="_blank" rel="noreferrer" className="text-blue-600 hover:underline">bot.zapps.me</a> và tạo một Bot mới.</p>
                </div>
              </div>
              <div className="flex gap-3">
                <div className="w-6 h-6 rounded-full bg-blue-100 text-blue-700 flex items-center justify-center flex-shrink-0 font-bold">2</div>
                <div>
                  <p>Copy <strong>Bot Token</strong> từ trang quản lý Bot Zalo, điền vào ô bên trên rồi nhấn Lưu. Hệ thống sẽ tự động cấu hình Webhook cho Bot của bạn.</p>
                </div>
              </div>
              <div className="flex gap-3">
                <div className="w-6 h-6 rounded-full bg-blue-100 text-blue-700 flex items-center justify-center flex-shrink-0 font-bold">3</div>
                <div>
                  <p>Tạo một Group Chat Zalo, <strong>đặt tên group theo đúng cú pháp</strong> <code>&lt;Tên Phòng&gt; &lt;Tên Nhà&gt;</code> (VD: <code>P101 679QT</code>) và thêm Bot vào nhóm.</p>
                  <p className="text-muted-foreground mt-1">Hệ thống sẽ tự động nhận diện và liên kết Group Zalo với phòng tương ứng.</p>
                </div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

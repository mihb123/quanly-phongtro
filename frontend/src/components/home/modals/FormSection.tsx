import type { ReactNode } from 'react'

// Khối thông tin lớn của form: chỉ in đậm tiêu đề, phân tách bằng đường kẻ mảnh (không tô nền).
export function FormSection({ title, hint, children }: { title: string; hint?: string; children: ReactNode }) {
  return (
    <section className="border-t border-border/60 pt-5 first:border-t-0 first:pt-0">
      <h3 className="text-sm font-semibold text-foreground">{title}</h3>
      {hint ? <p className="mt-0.5 text-xs text-muted-foreground">{hint}</p> : null}
      <div className="mt-3 space-y-4">{children}</div>
    </section>
  )
}

// Nhóm nhỏ bên trong một FormSection: nhãn nhạt màu, không in đậm để không tranh với tiêu đề khối.
export function FormSubGroup({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="space-y-2">
      <span className="block text-sm font-medium text-muted-foreground">{label}</span>
      {children}
    </div>
  )
}

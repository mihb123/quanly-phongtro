export function pageAudit({ mobile, phase }) {
  const vw = window.innerWidth
  const vh = window.innerHeight
  const issues = []
  const all = [...document.body.querySelectorAll('*')].filter((el) => !el.closest('[data-ui-shot-overlay]'))
  const styleCache = new Map()
  const css = (el) => {
    let s = styleCache.get(el)
    if (!s) { s = getComputedStyle(el); styleCache.set(el, s) }
    return s
  }
  const rectOf = (r) => ({ x: Math.round(r.left), y: Math.round(r.top), w: Math.round(r.width), h: Math.round(r.height) })
  const isShown = (el) => {
    const r = el.getBoundingClientRect()
    if (r.width <= 2 || r.height <= 2) return false
    if (el.checkVisibility && !el.checkVisibility({ opacityProperty: true, visibilityProperty: true, contentVisibilityAuto: true })) return false
    const s = css(el)
    if (s.clipPath && s.clipPath.startsWith('inset(50%')) return false
    if (s.clip && s.clip !== 'auto') return false
    return true
  }
  const owners = (el) => {
    const names = []
    let node = el
    while (node && !names.length) {
      const key = Object.keys(node).find((k) => k.startsWith('__reactFiber$'))
      if (key) {
        let fiber = node[key]
        while (fiber && names.length < 4) {
          const t = fiber.type
          const name = typeof t === 'function' ? (t.displayName || t.name)
            : t && typeof t === 'object' ? (t.displayName || t.render?.displayName || t.render?.name || t.type?.name) : null
          if (name && /^[A-Z]/.test(name) && !/(Context|Provider)$/.test(name) && !names.includes(name)) names.push(name)
          fiber = fiber.return
        }
      }
      node = node.parentElement
    }
    return names
  }
  const describe = (el, extraRect) => {
    const parts = [el.tagName.toLowerCase()]
    if (el.id) parts.push(`#${el.id}`)
    const slot = el.getAttribute('data-slot')
    if (slot) parts.push(`[data-slot=${slot}]`)
    const role = el.getAttribute('role')
    if (role) parts.push(`[role=${role}]`)
    const cls = typeof el.className === 'string' ? el.className.trim().split(/\s+/).filter(Boolean).slice(0, 4) : []
    if (cls.length) parts.push(`.${cls.join('.')}`)
    const holder = el.parentElement?.closest('[data-slot], [id]')
    const text = (el.innerText || el.getAttribute('aria-label') || el.getAttribute('placeholder') || '').replace(/\s+/g, ' ').trim()
    return {
      selector: parts.join(''),
      within: holder ? `${holder.tagName.toLowerCase()}${holder.id ? `#${holder.id}` : ''}${holder.getAttribute('data-slot') ? `[data-slot=${holder.getAttribute('data-slot')}]` : ''}` : null,
      text: text.slice(0, 80),
      components: owners(el),
      rect: rectOf(extraRect || el.getBoundingClientRect()),
    }
  }
  const add = (level, code, message, items = [], extra = {}) => issues.push({ level, code, message, items, ...extra })
  const ownText = (el) => [...el.childNodes].some((n) => n.nodeType === 3 && n.textContent.trim())
  const textRect = (el) => {
    const range = document.createRange()
    let box = null
    for (const n of el.childNodes) {
      if (n.nodeType !== 3 || !n.textContent.trim()) continue
      range.selectNodeContents(n)
      for (const r of range.getClientRects()) {
        if (r.width < 1 || r.height < 1) continue
        box = box ? { left: Math.min(box.left, r.left), top: Math.min(box.top, r.top), right: Math.max(box.right, r.right), bottom: Math.max(box.bottom, r.bottom) } : { left: r.left, top: r.top, right: r.right, bottom: r.bottom }
      }
    }
    return box ? { ...box, width: box.right - box.left, height: box.bottom - box.top } : null
  }
  const inter = (a, b) => {
    const w = Math.min(a.right, b.right) - Math.max(a.left, b.left)
    const h = Math.min(a.bottom, b.bottom) - Math.max(a.top, b.top)
    return w > 0 && h > 0 ? w * h : 0
  }
  const isFixedLayer = (el) => {
    for (let n = el; n && n !== document.body; n = n.parentElement) {
      const p = css(n).position
      if (p === 'fixed' || p === 'sticky') return n
    }
    return null
  }
  const visibleAll = all.filter(isShown)
  const POPUP = '[role=dialog], [role=alertdialog], [role=menu], [role=listbox], [data-slot=popover-content], [data-slot=select-content], [data-slot=dropdown-menu-content]'
  const modals = visibleAll.filter((el) => el.matches('[role=dialog], [role=alertdialog]') && el.getAttribute('aria-modal') !== 'false' && !el.parentElement?.closest('[role=dialog], [role=alertdialog]'))
  const shown = modals.length ? visibleAll.filter((el) => el.closest(POPUP)) : visibleAll
  const textEls = shown.filter(ownText)
  const textBoxes = textEls.map((el) => ({ el, box: textRect(el) })).filter((t) => t.box)
  const INTERACTIVE = 'a[href], button, input:not([type=hidden]), select, textarea, [role=button], [role=tab], [role=checkbox], [role=switch], [role=radio], [role=menuitem], [role=combobox], [role=option]'

  const fixedLayers = [...new Set(shown.map((el) => {
    const s = css(el)
    return s.position === 'fixed' || s.position === 'sticky' ? el : null
  }).filter(Boolean))].filter((el) => {
    const parentLayer = el.parentElement && isFixedLayer(el.parentElement)
    if (parentLayer) return false
    const r = el.getBoundingClientRect()
    return r.width * r.height < vw * vh * 0.5
  })

  if (phase === 'top') {
    if (document.body.innerText.trim().length < 10) add('error', 'page/blank', 'Trang gần như trống (không có nội dung chữ).')

    const busy = [...document.querySelectorAll('[data-slot="skeleton"], .animate-pulse, [aria-busy="true"]')].filter(isShown)
    if (busy.length) add('warn', 'page/still-loading', `Còn ${busy.length} skeleton/loading sau khi chờ ổn định — ảnh có thể chưa phản ánh dữ liệu thật.`, busy.slice(0, 3).map((el) => describe(el)))

    const docOverflow = Math.ceil(document.documentElement.scrollWidth) - vw
    const clippedAncestor = (el) => {
      for (let n = el.parentElement; n && n !== document.documentElement; n = n.parentElement) {
        const s = css(n)
        if (['hidden', 'clip', 'auto', 'scroll'].includes(s.overflowX) && n.getBoundingClientRect().right <= vw + 1) return n
      }
      return null
    }
    const outside = shown.filter((el) => {
      const r = el.getBoundingClientRect()
      return r.right > vw + 1 || r.left < -1
    })
    if (docOverflow > 1) {
      const causing = outside.filter((el) => !clippedAncestor(el))
      const set = new Set(causing)
      const origins = causing.filter((el) => !set.has(el.parentElement)).slice(0, 6)
      add('error', 'layout/overflow-x', `Trang tràn ngang ${docOverflow}px (scrollWidth ${vw + docOverflow} > viewport ${vw}) — người dùng phải vuốt ngang.`, origins.map((el) => describe(el)), { overflowPx: docOverflow })
    }
    const cut = outside.filter((el) => {
      const anc = clippedAncestor(el)
      if (!anc || ['auto', 'scroll'].includes(css(anc).overflowX)) return false
      if (!(ownText(el) || el.matches(INTERACTIVE))) return false
      const r = el.getBoundingClientRect()
      const a = anc.getBoundingClientRect()
      const visible = inter(r, a)
      return visible > 0 && visible < r.width * r.height * 0.95
    })
    if (cut.length) add('error', 'layout/cut-at-edge', `${cut.length} phần tử bị cắt ở mép màn hình bởi khung overflow:hidden.`, cut.slice(0, 6).map((el) => describe(el)))

    const scrollers = shown.filter((el) => {
      const s = css(el)
      return ['auto', 'scroll'].includes(s.overflowX) && el.scrollWidth > el.clientWidth + 1 && el.clientWidth > 0
    })
    if (scrollers.length && mobile) {
      add('info', 'layout/horizontal-scroller', `${scrollers.length} vùng cuộn ngang bên trong (bảng/tab). Trên mobile bảng nên chuyển sang card nếu nội dung chính bị ẩn.`,
        scrollers.slice(0, 4).map((el) => ({ ...describe(el), scrollWidth: el.scrollWidth, clientWidth: el.clientWidth })))
    }

    const clipped = []
    const truncated = []
    const spill = []
    for (const el of textEls) {
      const s = css(el)
      const hiddenX = ['hidden', 'clip'].includes(s.overflowX)
      const hiddenY = ['hidden', 'clip'].includes(s.overflowY)
      if (el.clientWidth > 0 && el.scrollWidth > el.clientWidth + 1) {
        if (hiddenX) (s.textOverflow === 'ellipsis' ? truncated : clipped).push(el)
        else if (s.overflowX === 'visible' && s.display !== 'inline') spill.push(el)
      }
      if (hiddenY && el.clientHeight > 0 && el.scrollHeight > el.clientHeight + 2) {
        const clamp = s.webkitLineClamp && s.webkitLineClamp !== 'none'
        if (!truncated.includes(el) && !clipped.includes(el)) (clamp ? truncated : clipped).push(el)
      }
    }
    for (const { el, box } of textBoxes) {
      if (clipped.includes(el) || truncated.includes(el)) continue
      for (let n = el.parentElement; n && n !== document.body; n = n.parentElement) {
        const s = css(n)
        if (['auto', 'scroll'].includes(s.overflowX) || ['auto', 'scroll'].includes(s.overflowY)) break
        if (!['hidden', 'clip'].includes(s.overflowX) && !['hidden', 'clip'].includes(s.overflowY)) continue
        const a = n.getBoundingClientRect()
        const area = box.width * box.height
        const seen = inter(box, a)
        if (seen > 0 && seen < area * 0.9) clipped.push(el)
        break
      }
    }
    if (clipped.length) add('error', 'text/clipped', `${clipped.length} đoạn chữ bị cắt mất (overflow hidden, không có dấu …).`, clipped.slice(0, 6).map((el) => ({ ...describe(el), fullText: el.textContent.trim().slice(0, 120) })))
    if (spill.length) add('warn', 'text/spill', `${spill.length} đoạn chữ tràn ra ngoài khung chứa (chữ dài không xuống dòng).`, spill.slice(0, 6).map((el) => describe(el)))
    if (truncated.length) add('info', 'text/truncated', `${truncated.length} đoạn chữ bị rút gọn bằng … — kiểm tra thông tin quan trọng không bị mất.`, truncated.slice(0, 6).map((el) => ({ ...describe(el), fullText: el.textContent.trim().slice(0, 120) })))

    const pieces = [
      ...textBoxes.filter(({ el }) => !isFixedLayer(el)).map(({ el, box }) => ({ el, box })),
      ...shown.filter((el) => el.matches(INTERACTIVE) && !isFixedLayer(el) && !el.innerText?.trim()).map((el) => ({ el, box: el.getBoundingClientRect() })),
    ].slice(0, 1500)
    const overlaps = []
    for (let i = 0; i < pieces.length && overlaps.length < 8; i++) {
      for (let j = i + 1; j < pieces.length && overlaps.length < 8; j++) {
        const a = pieces[i]
        const b = pieces[j]
        if (a.el.contains(b.el) || b.el.contains(a.el)) continue
        const area = inter(a.box, b.box)
        if (area <= 12) continue
        const minArea = Math.min(a.box.width * a.box.height, b.box.width * b.box.height)
        if (area < minArea * 0.25) continue
        const control = (el) => el.matches('input, textarea, select, [role=combobox]')
        if ((control(a.el) || control(b.el)) && Math.abs(area - minArea) < 2) continue
        const ctlA = a.el.closest('button, a, label, [role=button]')
        if (ctlA && ctlA === b.el.closest('button, a, label, [role=button]')) continue
        overlaps.push([a, b])
      }
    }
    if (overlaps.length) add('warn', 'layout/overlap', `${overlaps.length} cặp chữ/nút chồng lên nhau.`, overlaps.flatMap(([a, b]) => [describe(a.el, a.box), describe(b.el, b.box)]))

    const dialogs = shown.filter((el) => el.matches('[role=dialog], [role=alertdialog], [data-slot$="-content"][data-open], [data-slot=popover-content], [data-slot=dropdown-menu-content], [data-slot=select-content]'))
    const offscreen = dialogs.filter((el) => {
      const r = el.getBoundingClientRect()
      const out = r.bottom > vh + 1 || r.top < -1 || r.right > vw + 1 || r.left < -1
      const scrollable = ['auto', 'scroll'].includes(css(el).overflowY) && el.clientHeight <= vh
      return out && !scrollable
    })
    if (offscreen.length) add('error', 'overlay/offscreen', `${offscreen.length} dialog/popover vượt ra ngoài màn hình và không cuộn được.`, offscreen.map((el) => describe(el)))

    const broken = [...document.images].filter((img) => img.complete && img.naturalWidth === 0 && isShown(img))
    if (broken.length) add('error', 'media/broken-image', `${broken.length} ảnh không tải được.`, broken.slice(0, 5).map((img) => ({ ...describe(img), src: img.currentSrc || img.src })))

    if (mobile) {
      const small = shown.filter((el) => {
        if (!el.matches(INTERACTIVE) || el.closest('[aria-hidden=true]')) return false
        if (el.matches('input[type=checkbox], input[type=radio]') || el.getAttribute('role') === 'checkbox') {
          const label = el.closest('label') || (el.id && document.querySelector(`label[for="${CSS.escape(el.id)}"]`))
          if (label && label.getBoundingClientRect().height >= 24) return false
        }
        if (el.matches('a') && el.parentElement && ownText(el.parentElement)) return false
        const r = el.getBoundingClientRect()
        return r.width < 24 || r.height < 24
      })
      if (small.length) add('warn', 'touch/small-target', `${small.length} nút/vùng bấm nhỏ hơn 24×24px (khuyến nghị ≥ 40px trên mobile).`, small.slice(0, 6).map((el) => describe(el)))

      const tiny = textBoxes.filter(({ el }) => parseFloat(css(el).fontSize) < 12)
      if (tiny.length) add('warn', 'text/too-small', `${tiny.length} đoạn chữ nhỏ hơn 12px trên mobile.`, tiny.slice(0, 6).map(({ el, box }) => ({ ...describe(el, box), fontSize: css(el).fontSize })))
    }

    const canvas = document.createElement('canvas')
    canvas.width = canvas.height = 1
    const ctx = canvas.getContext('2d', { willReadFrequently: true })
    const rgba = (color) => {
      ctx.clearRect(0, 0, 1, 1)
      ctx.fillStyle = '#000'
      ctx.fillStyle = color
      ctx.fillRect(0, 0, 1, 1)
      const [r, g, b, a] = ctx.getImageData(0, 0, 1, 1).data
      return [r, g, b, a / 255]
    }
    const over = (top, bottom) => {
      const a = top[3] + bottom[3] * (1 - top[3])
      if (a === 0) return [0, 0, 0, 0]
      return [0, 1, 2].map((i) => (top[i] * top[3] + bottom[i] * bottom[3] * (1 - top[3])) / a).concat(a)
    }
    const lum = ([r, g, b]) => {
      const f = (v) => { v /= 255; return v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4 }
      return 0.2126 * f(r) + 0.7152 * f(g) + 0.0722 * f(b)
    }
    const backgroundOf = (el) => {
      const layers = []
      for (let n = el; n; n = n.parentElement) {
        const s = css(n)
        if (s.backgroundImage && s.backgroundImage !== 'none') return null
        const c = rgba(s.backgroundColor)
        if (c[3] > 0) layers.push(c)
        if (c[3] >= 1) break
      }
      let bg = [255, 255, 255, 1]
      if (!layers.length || layers[layers.length - 1][3] < 1) bg = rgba(css(document.documentElement).backgroundColor)
      if (bg[3] < 1) bg = [255, 255, 255, 1]
      for (let i = layers.length - 1; i >= 0; i--) bg = over(layers[i], bg)
      return bg
    }
    const lowContrast = []
    for (const { el, box } of textBoxes) {
      if (el.closest('[disabled], [aria-disabled=true], [data-disabled]')) continue
      let faded = false
      for (let n = el; n; n = n.parentElement) if (parseFloat(css(n).opacity) < 1) { faded = true; break }
      if (faded) continue
      const bg = backgroundOf(el)
      if (!bg) continue
      const fg = over(rgba(css(el).color), bg)
      const l1 = lum(fg)
      const l2 = lum(bg)
      const ratio = (Math.max(l1, l2) + 0.05) / (Math.min(l1, l2) + 0.05)
      const size = parseFloat(css(el).fontSize)
      const large = size >= 24 || (size >= 18.6 && Number(css(el).fontWeight) >= 700)
      const need = large ? 3 : 4.5
      if (ratio < need) lowContrast.push({ el, box, ratio, need, fg, bg })
    }
    const invisible = lowContrast.filter((c) => c.ratio < 2)
    const weak = lowContrast.filter((c) => c.ratio >= 2)
    const hex = (c) => `#${c.slice(0, 3).map((v) => Math.round(v).toString(16).padStart(2, '0')).join('')}`
    const contrastItem = (c) => ({ ...describe(c.el, c.box), ratio: Math.round(c.ratio * 100) / 100, need: c.need, fg: hex(c.fg), bg: hex(c.bg) })
    if (invisible.length) add('error', 'text/invisible', `${invisible.length} đoạn chữ gần như không đọc được (tương phản < 2:1).`, invisible.slice(0, 6).map(contrastItem))
    if (weak.length) add('warn', 'text/low-contrast', `${weak.length} đoạn chữ tương phản dưới chuẩn WCAG AA.`, weak.sort((a, b) => a.ratio - b.ratio).slice(0, 6).map(contrastItem))
  }

  const scroller = (() => {
    const docEl = document.scrollingElement || document.documentElement
    let best = docEl.scrollHeight > vh + 1 ? { el: docEl, area: vw * vh } : null
    for (const el of shown) {
      const s = css(el)
      if (!['auto', 'scroll', 'overlay'].includes(s.overflowY)) continue
      if (el.scrollHeight <= el.clientHeight + 1 || el.clientHeight < vh * 0.4) continue
      const area = el.clientWidth * el.clientHeight
      if (!best || area > best.area) best = { el, area }
    }
    return best?.el || null
  })()
  const atBottom = !scroller || scroller.scrollTop + scroller.clientHeight >= scroller.scrollHeight - 2
  const atTop = !scroller || scroller.scrollTop <= 1
  const checkEdges = []
  if (phase === 'top' && atTop) checkEdges.push('top')
  if (atBottom) checkEdges.push('bottom')
  const covered = []
  for (const layer of fixedLayers) {
    const r = layer.getBoundingClientRect()
    const edge = (r.top + r.bottom) / 2 < vh / 2 ? 'top' : 'bottom'
    if (!checkEdges.includes(edge)) continue
    if (css(layer).position === 'sticky' && edge === 'top' && atTop) continue
    for (const { el, box } of textBoxes) {
      if (layer.contains(el) || isFixedLayer(el)) continue
      const area = inter(box, r)
      if (area < box.width * box.height * 0.3) continue
      const cx = (Math.max(box.left, r.left) + Math.min(box.right, r.right)) / 2
      const cy = (Math.max(box.top, r.top) + Math.min(box.bottom, r.bottom)) / 2
      const hit = document.elementFromPoint(cx, cy)
      if (hit && layer.contains(hit)) covered.push({ el, box, layer })
    }
  }
  if (covered.length) {
    add('error', 'layout/covered-by-fixed', `${covered.length} nội dung bị thanh cố định (header/bottom nav/nút nổi) che khi cuộn ${phase === 'top' ? 'ở đầu' : 'tới cuối'} trang — thiếu padding cho vùng cuộn.`,
      covered.slice(0, 6).flatMap((c) => [describe(c.el, c.box), { ...describe(c.layer), role: 'cover' }]))
  }
  if (phase === 'bottom' && scroller && scroller.scrollTop > 0) {
    const floor = fixedLayers.reduce((min, layer) => {
      const r = layer.getBoundingClientRect()
      return r.top > vh / 2 ? Math.min(min, r.top) : min
    }, vh)
    const contentBottom = shown.reduce((max, el) => {
      if (isFixedLayer(el) || !(ownText(el) || el.matches(INTERACTIVE) || el.matches('img, svg, canvas, video'))) return max
      return Math.max(max, el.getBoundingClientRect().bottom)
    }, 0)
    const gap = Math.round(floor - contentBottom)
    if (gap > vh * 0.3) add('warn', 'layout/empty-tail', `Cuộn tới cuối còn ${gap}px trống phía dưới nội dung (thừa padding/min-height làm trang cuộn được vào khoảng trắng).`, [], { gapPx: gap })
  }

  return {
    issues,
    viewport: { width: vw, height: vh },
    theme: document.documentElement.classList.contains('dark') ? 'dark' : 'light',
    path: location.pathname + location.search,
    scroller: scroller ? { tag: scroller === (document.scrollingElement || document.documentElement) ? 'document' : describe(scroller).selector, scrollTop: Math.round(scroller.scrollTop), scrollHeight: scroller.scrollHeight, clientHeight: scroller.clientHeight } : null,
  }
}


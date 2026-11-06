package thuchi

const (
	CatAnUong          = "an_uong"
	CatBuaLon          = "bua_lon"
	CatMuaSam          = "mua_sam"
	CatMyPham          = "my_pham"
	CatLamDep          = "lam_dep"
	CatDiLai           = "di_lai"
	CatXangXe          = "xang_xe"
	CatThuocMen        = "thuoc_men"
	CatHocTap          = "hoc_tap"
	CatDichVuOnline    = "dich_vu_online"
	CatGiaiTri         = "giai_tri"
	CatHoaDon          = "hoa_don"
	CatNhaCua          = "nha_cua"
	CatChuyenNguoiKhac = "chuyen_nguoi_khac"
	CatPhiNganHang     = "phi_ngan_hang"
	CatKhacChi         = "khac_chi"
	CatLuong           = "luong"
	CatTienPhong       = "tien_phong"
	CatDuocChuyen      = "duoc_chuyen"
	CatKhacThu         = "khac_thu"
)

type DefaultCategory struct {
	Key   string
	Name  string
	Icon  string
	Color string
	Kind  string
}

var DefaultCategories = []DefaultCategory{
	{CatAnUong, "Ăn uống", "food", "#f97316", "expense"},
	{CatBuaLon, "Bữa lớn", "restaurant", "#ef4444", "expense"},
	{CatMuaSam, "Mua sắm", "shopping", "#ec4899", "expense"},
	{CatMyPham, "Mỹ phẩm", "cosmetic", "#d946ef", "expense"},
	{CatLamDep, "Làm đẹp", "beauty", "#a855f7", "expense"},
	{CatDiLai, "Đi lại", "car", "#3b82f6", "expense"},
	{CatXangXe, "Xăng xe", "fuel", "#0ea5e9", "expense"},
	{CatThuocMen, "Thuốc men", "medicine", "#14b8a6", "expense"},
	{CatHocTap, "Học tập", "book", "#22c55e", "expense"},
	{CatDichVuOnline, "Dịch vụ online", "cloud", "#6366f1", "expense"},
	{CatGiaiTri, "Giải trí", "game", "#8b5cf6", "expense"},
	{CatHoaDon, "Hoá đơn & tiện ích", "bill", "#eab308", "expense"},
	{CatNhaCua, "Nhà cửa", "home", "#84cc16", "expense"},
	{CatChuyenNguoiKhac, "Chuyển cho người khác", "send", "#64748b", "expense"},
	{CatPhiNganHang, "Phí ngân hàng", "bank", "#78716c", "expense"},
	{CatKhacChi, "Khác", "more", "#94a3b8", "expense"},
	{CatLuong, "Lương", "salary", "#16a34a", "income"},
	{CatTienPhong, "Tiền phòng", "key", "#0d9488", "income"},
	{CatDuocChuyen, "Được chuyển", "receive", "#2563eb", "income"},
	{CatKhacThu, "Khác", "more", "#94a3b8", "income"},
}

type MerchantEntry struct {
	Keywords []string
	Name     string
	Category string
}

var MerchantDictionary = []MerchantEntry{
	{[]string{"GRABFOOD", "GRAB FOOD"}, "GrabFood", CatAnUong},
	{[]string{"SHOPEEFOOD", "SHOPEE FOOD"}, "ShopeeFood", CatAnUong},
	{[]string{"BEFOOD", "BE FOOD"}, "beFood", CatAnUong},
	{[]string{"GRAB"}, "Grab", CatDiLai},
	{[]string{"XANH SM", "XANHSM"}, "Xanh SM", CatDiLai},
	{[]string{"BE GROUP", "BEGROUP", "BE JSC"}, "Be", CatDiLai},
	{[]string{"VETC", "EPASS", "VDTC"}, "Phí đường bộ", CatDiLai},
	{[]string{"PETROLIMEX", "PVOIL", "XANG DAU", "PETRO"}, "Xăng dầu", CatXangXe},
	{[]string{"HIGHLANDS", "HIGHLAND COFFEE"}, "Highlands Coffee", CatAnUong},
	{[]string{"STARBUCKS"}, "Starbucks", CatAnUong},
	{[]string{"PHUC LONG"}, "Phúc Long", CatAnUong},
	{[]string{"KATINAT"}, "Katinat", CatAnUong},
	{[]string{"COFFEE HOUSE", "THE COFFEE HOUSE"}, "The Coffee House", CatAnUong},
	{[]string{"KFC"}, "KFC", CatAnUong},
	{[]string{"LOTTERIA"}, "Lotteria", CatAnUong},
	{[]string{"JOLLIBEE"}, "Jollibee", CatAnUong},
	{[]string{"MCDONALD"}, "McDonald's", CatAnUong},
	{[]string{"PIZZA"}, "Pizza", CatBuaLon},
	{[]string{"CIRCLE K", "CIRCLEK"}, "Circle K", CatAnUong},
	{[]string{"FAMILYMART", "FAMILY MART"}, "FamilyMart", CatAnUong},
	{[]string{"GS25"}, "GS25", CatAnUong},
	{[]string{"7 ELEVEN", "7ELEVEN"}, "7-Eleven", CatAnUong},
	{[]string{"WINMART", "WIN MART", "VINMART"}, "WinMart", CatMuaSam},
	{[]string{"BACH HOA XANH", "BACHHOAXANH"}, "Bách Hoá Xanh", CatMuaSam},
	{[]string{"CO OP", "COOPMART", "CO OPMART"}, "Co.opmart", CatMuaSam},
	{[]string{"AEON"}, "AEON", CatMuaSam},
	{[]string{"LOTTE MART"}, "Lotte Mart", CatMuaSam},
	{[]string{"SHOPEE", "SHOPEEPAY"}, "Shopee", CatMuaSam},
	{[]string{"LAZADA"}, "Lazada", CatMuaSam},
	{[]string{"TIKI"}, "Tiki", CatMuaSam},
	{[]string{"TIKTOK SHOP", "TIKTOKSHOP"}, "TikTok Shop", CatMuaSam},
	{[]string{"UNIQLO"}, "Uniqlo", CatMuaSam},
	{[]string{"GUARDIAN"}, "Guardian", CatMyPham},
	{[]string{"WATSONS"}, "Watsons", CatMyPham},
	{[]string{"HASAKI"}, "Hasaki", CatMyPham},
	{[]string{"PHARMACITY"}, "Pharmacity", CatThuocMen},
	{[]string{"LONG CHAU"}, "Long Châu", CatThuocMen},
	{[]string{"AN KHANG"}, "An Khang", CatThuocMen},
	{[]string{"NETFLIX"}, "Netflix", CatDichVuOnline},
	{[]string{"SPOTIFY"}, "Spotify", CatDichVuOnline},
	{[]string{"YOUTUBE", "GOOGLE YOUTUBE"}, "YouTube", CatDichVuOnline},
	{[]string{"APPLE COM", "APPLE COM BILL", "ITUNES"}, "Apple", CatDichVuOnline},
	{[]string{"GOOGLE"}, "Google", CatDichVuOnline},
	{[]string{"OPENAI", "CHATGPT"}, "OpenAI", CatDichVuOnline},
	{[]string{"ANTHROPIC", "CLAUDE AI"}, "Anthropic", CatDichVuOnline},
	{[]string{"CGV"}, "CGV", CatGiaiTri},
	{[]string{"LOTTE CINEMA"}, "Lotte Cinema", CatGiaiTri},
	{[]string{"STEAM"}, "Steam", CatGiaiTri},
	{[]string{"EVN", "TIEN DIEN"}, "Tiền điện", CatHoaDon},
	{[]string{"TIEN NUOC", "CAP NUOC"}, "Tiền nước", CatHoaDon},
	{[]string{"VIETTEL"}, "Viettel", CatHoaDon},
	{[]string{"VNPT"}, "VNPT", CatHoaDon},
	{[]string{"FPT TELECOM", "FPT"}, "FPT", CatHoaDon},
	{[]string{"MOBIFONE"}, "MobiFone", CatHoaDon},
	{[]string{"VINAPHONE"}, "VinaPhone", CatHoaDon},
	{[]string{"PHI DUY TRI", "PHI SMS", "PHI QUAN LY TAI KHOAN", "PHI CHUYEN TIEN"}, "Phí ngân hàng", CatPhiNganHang},
}

var walletKeywords = map[string][]string{
	"MOMO":         {"MOMO", "M SERVICE"},
	"ZALOPAY":      {"ZALOPAY", "ZALO PAY"},
	"VIETTELMONEY": {"VIETTELPAY", "VIETTEL MONEY"},
	"SHOPEEPAY":    {"SHOPEEPAY"},
}

var refundKeywords = []string{"HOAN TIEN", "HOAN TRA", "REFUND", "HUY GIAO DICH", "TRA LAI TIEN", "CASHBACK"}

var cardPaymentKeywords = []string{"THANH TOAN THE TIN DUNG", "TT THE TIN DUNG", "TT THE TD", "THANH TOAN DU NO", "CREDIT CARD PAYMENT", "TRA NO THE"}

var BankNames = map[string]string{
	"TPB":     "TPBank",
	"VCB":     "Vietcombank",
	"MB":      "MB",
	"ACB":     "ACB",
	"TCB":     "Techcombank",
	"CTG":     "VietinBank",
	"STB":     "Sacombank",
	"TIMO":    "Timo",
	"CAKE":    "Cake",
	"MOMO":    "MoMo",
	"ZALOPAY": "ZaloPay",
	"BIDV":    "BIDV",
	"VPB":     "VPBank",
	"OCB":     "OCB",
	"MSB":     "MSB",
	"HDB":     "HDBank",
	"VIB":     "VIB",
	"KLB":     "KienlongBank",
}

func gatewayToBankCode(gateway string) string {
	switch Normalize(gateway) {
	case "TPBANK", "TPB":
		return "TPB"
	case "VIETCOMBANK", "VCB":
		return "VCB"
	case "MBBANK", "MB", "MB BANK":
		return "MB"
	case "ACB":
		return "ACB"
	case "TECHCOMBANK", "TCB":
		return "TCB"
	case "VIETINBANK", "CTG":
		return "CTG"
	case "SACOMBANK", "STB":
		return "STB"
	case "BIDV":
		return "BIDV"
	case "VPBANK", "VPB":
		return "VPB"
	case "OCB":
		return "OCB"
	case "MSB":
		return "MSB"
	case "HDBANK", "HDB":
		return "HDB"
	case "VIB":
		return "VIB"
	case "KIENLONGBANK", "KLB":
		return "KLB"
	default:
		return Normalize(gateway)
	}
}

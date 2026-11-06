// Barrel icon dùng chung của app.
// Mục đích: bind sẵn HugeiconsIcon với từng icon "ngữ nghĩa" có TÊN TRÙNG với lucide cũ,
// nhờ vậy migrate chỉ cần đổi nguồn import ('lucide-react' -> '@/components/icons') mà giữ nguyên JSX.
// Đổi/bộ icon về sau chỉ sửa duy nhất file này (map name -> hugeicons export).
import { HugeiconsIcon, type HugeiconsProps, type IconSvgElement } from '@hugeicons/react'
import {
  AlertCircleIcon,
  Alert02Icon,
  ArrowRight01Icon,
  ArrowLeft01Icon,
  Building03Icon,
  Building06Icon,
  Calendar03Icon,
  Tick02Icon,
  CheckmarkCircle02Icon,
  ArrowDown01Icon,
  ArrowUp01Icon,
  Clock01Icon,
  Copy01Icon,
  Dollar01Icon,
  Door01Icon,
  Download04Icon,
  DropletsIcon,
  Edit02Icon,
  PencilEdit02Icon,
  ExternalLinkIcon,
  EyeIcon,
  EyeOffIcon,
  File02Icon,
  Doc01Icon,
  FilterRemoveIcon,
  Home09Icon,
  Image02Icon,
  Key01Icon,
  DashboardSquare01Icon,
  Loading03Icon,
  LockIcon,
  Login03Icon,
  Logout03Icon,
  Mail01Icon,
  TelephoneIcon,
  PlusSignIcon,
  Invoice01Icon,
  ArrowReloadHorizontalIcon,
  FloppyDiskIcon,
  Navigation03Icon,
  Settings01Icon,
  Shield01Icon,
  Delete02Icon,
  TradeDownIcon,
  TradeUpIcon,
  Upload04Icon,
  UserIcon,
  UserCircleIcon,
  UserAdd01Icon,
  UserGroupIcon,
  Wallet01Icon,
  Cancel01Icon,
  FlashIcon,
  ZoomInAreaIcon,
  Search01Icon,
  Sun03Icon,
  Moon02Icon,
  ComputerIcon,
  PuzzleIcon,
  FileUploadIcon,
  BankIcon,
  Cash01Icon,
  CloudUploadIcon,
  CreditCardIcon,
  InboxIcon,
  LeftToRightListBulletIcon,
  Location01Icon,
  PieChartIcon,
  Tag01Icon,
  ArrowDataTransferHorizontalIcon,
  WifiDisconnected01Icon,
  Invoice03Icon,
  Book01Icon,
  Car01Icon,
  CloudIcon,
  Coffee01Icon,
  FileExportIcon,
  FuelStationIcon,
  GameController01Icon,
  Home01Icon,
  Medicine01Icon,
  Money01Icon,
  MoneyReceive01Icon,
  MoreHorizontalIcon,
  PerfumeIcon,
  SmartPhone01Icon,
  Restaurant02Icon,
  SentIcon,
  ShoppingBag01Icon,
  SparklesIcon,
  Target01Icon,
  Restaurant01Icon,
} from '@hugeicons/core-free-icons'

// Props của 1 icon: giống lucide (nhận className/size/onClick...), trừ `icon` đã bind sẵn.
export type IconProps = Omit<HugeiconsProps, 'icon'>

// Tạo 1 component icon từ svg object của hugeicons, giữ API gọi giống lucide.
function makeIcon(svg: IconSvgElement) {
  return function Icon(props: IconProps) {
    return <HugeiconsIcon icon={svg} {...props} />
  }
}

export const AlertCircle = makeIcon(AlertCircleIcon)
export const AlertTriangle = makeIcon(Alert02Icon)
export const ArrowRight = makeIcon(ArrowRight01Icon)
export const Building = makeIcon(Building03Icon)
export const Building2 = makeIcon(Building06Icon)
export const Calendar = makeIcon(Calendar03Icon)
export const Check = makeIcon(Tick02Icon)
export const CheckCircle2 = makeIcon(CheckmarkCircle02Icon)
export const ChevronDown = makeIcon(ArrowDown01Icon)
export const ChevronLeft = makeIcon(ArrowLeft01Icon)
export const ChevronRight = makeIcon(ArrowRight01Icon)
export const ChevronUp = makeIcon(ArrowUp01Icon)
export const Clock = makeIcon(Clock01Icon)
export const Copy = makeIcon(Copy01Icon)
export const DollarSign = makeIcon(Dollar01Icon)
export const DoorOpen = makeIcon(Door01Icon)
export const Download = makeIcon(Download04Icon)
export const Droplets = makeIcon(DropletsIcon)
export const Edit = makeIcon(Edit02Icon)
export const ExternalLink = makeIcon(ExternalLinkIcon)
export const Eye = makeIcon(EyeIcon)
export const EyeOff = makeIcon(EyeOffIcon)
export const FileIcon = makeIcon(File02Icon)
export const FileText = makeIcon(Doc01Icon)
export const FilterX = makeIcon(FilterRemoveIcon)
export const Home = makeIcon(Home09Icon)
export const ImageIcon = makeIcon(Image02Icon)
export const Key = makeIcon(Key01Icon)
export const LayoutDashboard = makeIcon(DashboardSquare01Icon)
export const Loader2 = makeIcon(Loading03Icon)
export const Lock = makeIcon(LockIcon)
export const LogIn = makeIcon(Login03Icon)
export const LogOut = makeIcon(Logout03Icon)
// Three-dots ngang (⋯) vẽ filled thủ công: glyph hugeicons dùng stroke r=1 quá mảnh,
// render ở 16px nhìn không rõ là ba dấu chấm. Bọc qua factory để giữ dạng
// `export const X = f()` như makeIcon (tránh rule react-refresh/only-export-components).
const makeFilledMoreHorizontal = () =>
  function MoreHorizontalGlyph({ className, ...props }: React.SVGProps<SVGSVGElement>) {
    return (
      <svg
        viewBox="0 0 24 24"
        width={24}
        height={24}
        fill="currentColor"
        aria-hidden="true"
        className={className}
        {...props}
      >
        <circle cx="5" cy="12" r="2" />
        <circle cx="12" cy="12" r="2" />
        <circle cx="19" cy="12" r="2" />
      </svg>
    )
  }
export const MoreHorizontal = makeFilledMoreHorizontal()
export const Mail = makeIcon(Mail01Icon)
export const Pencil = makeIcon(PencilEdit02Icon)
export const Phone = makeIcon(TelephoneIcon)
export const Plus = makeIcon(PlusSignIcon)
export const Receipt = makeIcon(Invoice01Icon)
export const RefreshCw = makeIcon(ArrowReloadHorizontalIcon)
export const Save = makeIcon(FloppyDiskIcon)
export const Send = makeIcon(Navigation03Icon)
export const Search = makeIcon(Search01Icon)
export const Settings = makeIcon(Settings01Icon)
export const Shield = makeIcon(Shield01Icon)
export const Trash2 = makeIcon(Delete02Icon)
export const TrendingDown = makeIcon(TradeDownIcon)
export const TrendingUp = makeIcon(TradeUpIcon)
export const Upload = makeIcon(Upload04Icon)
export const User = makeIcon(UserIcon)
export const UserCircle = makeIcon(UserCircleIcon)
export const UserPlus = makeIcon(UserAdd01Icon)
export const Users = makeIcon(UserGroupIcon)
export const Wallet = makeIcon(Wallet01Icon)
export const X = makeIcon(Cancel01Icon)
export const Zap = makeIcon(FlashIcon)
export const ZoomIn = makeIcon(ZoomInAreaIcon)
export const Sun = makeIcon(Sun03Icon)
export const Moon = makeIcon(Moon02Icon)
export const Monitor = makeIcon(ComputerIcon)
export const Puzzle = makeIcon(PuzzleIcon)
export const FileUp = makeIcon(FileUploadIcon)
export const Bank = makeIcon(BankIcon)
export const Cash = makeIcon(Cash01Icon)
export const CloudUpload = makeIcon(CloudUploadIcon)
export const CreditCard = makeIcon(CreditCardIcon)
export const Inbox = makeIcon(InboxIcon)
export const ListIcon = makeIcon(LeftToRightListBulletIcon)
export const MapPin = makeIcon(Location01Icon)
export const PieChart = makeIcon(PieChartIcon)
export const Tag = makeIcon(Tag01Icon)
export const TransferIcon = makeIcon(ArrowDataTransferHorizontalIcon)
export const WifiOff = makeIcon(WifiDisconnected01Icon)
export const Bill = makeIcon(Invoice03Icon)
export const Book = makeIcon(Book01Icon)
export const Car = makeIcon(Car01Icon)
export const Cloud = makeIcon(CloudIcon)
export const Coffee = makeIcon(Coffee01Icon)
export const FileExport = makeIcon(FileExportIcon)
export const Fuel = makeIcon(FuelStationIcon)
export const Game = makeIcon(GameController01Icon)
export const House = makeIcon(Home01Icon)
export const Medicine = makeIcon(Medicine01Icon)
export const Money = makeIcon(Money01Icon)
export const MoneyIn = makeIcon(MoneyReceive01Icon)
export const MoreDots = makeIcon(MoreHorizontalIcon)
export const Perfume = makeIcon(PerfumeIcon)
export const Phone2 = makeIcon(SmartPhone01Icon)
export const Restaurant = makeIcon(Restaurant02Icon)
export const SendMoney = makeIcon(SentIcon)
export const ShoppingBag = makeIcon(ShoppingBag01Icon)
export const Sparkles = makeIcon(SparklesIcon)
export const Target = makeIcon(Target01Icon)
export const Utensils = makeIcon(Restaurant01Icon)

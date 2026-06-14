import { Building, ChevronDown, Check } from 'lucide-react';

interface House {
  id: string;
  name: string;
}

interface HouseSelectDropdownProps {
  houses: House[];
  selectedHouseIds: string[];
  isOpen: boolean;
  onOpenChange: (isOpen: boolean) => void;
  onToggleHouse: (houseId: string) => void;
  onSelectAll: () => void;
}

export function HouseSelectDropdown({
  houses,
  selectedHouseIds,
  isOpen,
  onOpenChange,
  onToggleHouse,
  onSelectAll
}: HouseSelectDropdownProps) {
  return (
    <div className="relative w-full sm:w-auto">
      <button
        onClick={() => onOpenChange(!isOpen)}
        className="flex items-center justify-between w-full sm:w-48 bg-background hover:bg-accent hover:text-accent-foreground border border-input text-foreground h-10 px-3 sm:px-4 py-2 rounded-md text-sm font-medium shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring transition-colors cursor-pointer overflow-hidden"
      >
        <div className="flex items-center gap-2 overflow-hidden min-w-0">
          <Building className="w-4 h-4 text-muted-foreground shrink-0" />
          <span className="truncate whitespace-nowrap">
            {selectedHouseIds.length === houses.length && houses.length > 0
              ? "Tất cả các nhà"
              : selectedHouseIds.length === 0
              ? "Chọn nhà..."
              : selectedHouseIds.length === 1
              ? houses.find(h => h.id === selectedHouseIds[0])?.name
              : `Đã chọn ${selectedHouseIds.length} nhà`}
          </span>
        </div>
        <ChevronDown className={`w-4 h-4 text-muted-foreground transition-transform ${isOpen ? 'rotate-180' : ''}`} />
      </button>
      
      {isOpen && (
        <>
          <div 
            className="fixed inset-0 z-10" 
            onClick={() => onOpenChange(false)} 
          />
          <div className="absolute top-full left-0 mt-2 min-w-[16rem] w-max max-w-[90vw] sm:max-w-md bg-card border border-border rounded-xl shadow-lg z-20 py-2 animate-in fade-in zoom-in-95 duration-200">
            {houses.length > 1 && (
              <button
                onClick={onSelectAll}
                className="w-full flex items-center gap-3 px-4 py-2 hover:bg-secondary/50 transition-colors text-left font-bold text-primary"
              >
                <div className={`w-4 h-4 rounded-sm border flex items-center justify-center shrink-0 transition-colors ${selectedHouseIds.length === houses.length ? 'bg-primary border-primary text-primary-foreground' : 'border-primary/50'}`}>
                  {selectedHouseIds.length === houses.length && <Check className="w-3 h-3" />}
                </div>
                <span className="flex-1 break-words">Tất cả các nhà</span>
              </button>
            )}
            {houses.length > 1 && <div className="h-px bg-border/50 my-1 mx-2" />}
            <div className="max-h-64 overflow-y-auto">
              {houses.map(house => {
                const isSelected = selectedHouseIds.includes(house.id);
                return (
                  <button
                    key={house.id}
                    onClick={() => onToggleHouse(house.id)}
                    className="w-full flex items-start gap-3 px-4 py-2 hover:bg-secondary/50 transition-colors text-left"
                  >
                    <div className={`mt-0.5 w-4 h-4 rounded-sm border flex items-center justify-center shrink-0 transition-colors ${isSelected ? 'bg-primary border-primary text-primary-foreground' : 'border-input'}`}>
                      {isSelected && <Check className="w-3 h-3" />}
                    </div>
                    <span className={`font-medium flex-1 break-words ${isSelected ? 'text-foreground' : 'text-muted-foreground'}`}>
                      {house.name}
                    </span>
                  </button>
                );
              })}
            </div>
          </div>
        </>
      )}
    </div>
  );
}

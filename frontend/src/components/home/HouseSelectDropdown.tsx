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
    <div className="relative inline-flex items-center">
      <button
        onClick={() => onOpenChange(!isOpen)}
        className="flex items-center justify-between w-64 bg-secondary/30 border border-border/50 text-foreground py-2.5 px-4 rounded-xl font-bold shadow-sm focus:outline-none focus:ring-2 focus:ring-primary/50 transition-all cursor-pointer hover:bg-secondary/50"
      >
        <div className="flex items-center gap-2 overflow-hidden">
          <Building className="w-4 h-4 text-muted-foreground flex-shrink-0" />
          <span className="truncate">
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
          <div className="absolute top-full left-0 mt-2 w-64 bg-card border border-border rounded-xl shadow-lg z-20 py-2 animate-in fade-in zoom-in-95 duration-200">
            {houses.length > 1 && (
              <button
                onClick={onSelectAll}
                className="w-full flex items-center justify-between px-4 py-2 hover:bg-secondary/50 transition-colors text-left font-bold text-primary"
              >
                <span>Tất cả các nhà</span>
                {selectedHouseIds.length === houses.length && <Check className="w-4 h-4" />}
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
                    className="w-full flex items-center justify-between px-4 py-2 hover:bg-secondary/50 transition-colors text-left"
                  >
                    <span className={`font-medium ${isSelected ? 'text-foreground' : 'text-muted-foreground'}`}>
                      {house.name}
                    </span>
                    {isSelected && <Check className="w-4 h-4 text-primary" />}
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

import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';

export const HOUSE_BUTTON_LIMIT = 5;

interface HouseButtonGroupProps {
  houses: { id: string; name: string }[];
  isSelected: (houseId: string) => boolean;
  onSelect: (houseId: string) => void;
  allLabel: string;
  isAllSelected: boolean;
  onSelectAll: () => void;
  className?: string;
}

export function HouseButtonGroup({
  houses,
  isSelected,
  onSelect,
  allLabel,
  isAllSelected,
  onSelectAll,
  className,
}: HouseButtonGroupProps) {
  return (
    <div className={cn('flex flex-wrap gap-2', className)}>
      <Button
        type="button"
        variant={isAllSelected ? 'default' : 'outline'}
        aria-pressed={isAllSelected}
        onClick={onSelectAll}
        className="h-11 cursor-pointer sm:h-9"
      >
        {allLabel}
      </Button>
      {houses.map(house => {
        const active = !isAllSelected && isSelected(house.id);
        return (
          <Button
            key={house.id}
            type="button"
            variant={active ? 'default' : 'outline'}
            aria-pressed={active}
            onClick={() => onSelect(house.id)}
            className="h-11 max-w-full cursor-pointer sm:h-9"
          >
            <span className="truncate">{house.name}</span>
          </Button>
        );
      })}
    </div>
  );
}

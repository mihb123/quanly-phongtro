import { useState, useEffect } from 'react';
import { type House } from '@/api/house';
import { getAllInvoices } from '@/api/invoice';
import { readStorageRaw, writeStorageRaw } from '@/lib/storage';
import { useRoomStore } from '@/data/roomData';

export function useRecommendedHouse(houses: House[], period: string, fallbackHouseId?: string) {
  const [recommendedHouseId, setRecommendedHouseId] = useState<string>(fallbackHouseId || '');
  const [isLoading, setIsLoading] = useState(true);
  const { getRoomsByHouse } = useRoomStore();

  useEffect(() => {
    let isMounted = true;
    
    const findHouse = async () => {
      if (!houses || houses.length === 0) {
        if (isMounted) {
          setRecommendedHouseId('');
          setIsLoading(false);
        }
        return;
      }
      
      if (houses.length === 1) {
        if (isMounted) {
          setRecommendedHouseId(houses[0].id);
          setIsLoading(false);
        }
        return;
      }

      setIsLoading(true);

      const lastSelected = readStorageRaw('lastSelectedHouseId_Invoice');
      const candidateHouses = [...houses].sort((a, b) => {
        const dateA = a.created_at ? new Date(a.created_at).getTime() : 0;
        const dateB = b.created_at ? new Date(b.created_at).getTime() : 0;
        return dateB - dateA;
      });

      if (lastSelected) {
        const lastIdx = candidateHouses.findIndex(h => h.id === lastSelected);
        if (lastIdx > -1) {
          const [lastHouse] = candidateHouses.splice(lastIdx, 1);
          candidateHouses.unshift(lastHouse);
        }
      }

      let foundId = candidateHouses[0].id; // Default fallback to the highest priority one
      
      for (const house of candidateHouses) {
        try {
          const fetchedRooms = await getRoomsByHouse(house.id);
          const occupiedRooms = fetchedRooms.filter(r => r.status === 'OCCUPIED');
          
          if (occupiedRooms.length === 0) continue; // Skip houses with no occupied rooms? Or maybe consider them invoiced? We'll skip to find one with actual work.

          const periodInvoices = await getAllInvoices({ house_id: house.id, period });
          
          let hasUninvoiced = false;
          for (const room of occupiedRooms) {
            if (!periodInvoices.some(inv => inv.room_id === room.id)) {
              hasUninvoiced = true;
              break;
            }
          }

          if (hasUninvoiced) {
            foundId = house.id;
            break;
          }
        } catch (error) {
          console.error("Error checking house status", house.id, error);
        }
      }

      if (isMounted) {
        setRecommendedHouseId(foundId);
        setIsLoading(false);
      }
    };

    findHouse();

    return () => { isMounted = false; };
  }, [houses, period, getRoomsByHouse]);

  const saveSelectedHouse = (id: string) => {
    writeStorageRaw('lastSelectedHouseId_Invoice', id);
  };

  return { recommendedHouseId, isLoading, saveSelectedHouse };
}

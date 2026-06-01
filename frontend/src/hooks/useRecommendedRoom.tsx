import { useState, useEffect } from 'react';
import { type Room } from '@/api/room';
import { getInvoices } from '@/api/invoice';

export function useRecommendedRoom(houseId: string, period: string, rooms: Room[]) {
  const [recommendedRoomId, setRecommendedRoomId] = useState<string>('');
  const [isLoadingRoom, setIsLoadingRoom] = useState(false);

  useEffect(() => {
    let isMounted = true;
    
    const findRoom = async () => {
      if (!houseId || !period || !rooms || rooms.length === 0) {
        if (isMounted) {
          setRecommendedRoomId('');
          setIsLoadingRoom(false);
        }
        return;
      }

      setIsLoadingRoom(true);

      try {
        const occupiedRooms = rooms.filter(r => r.status === 'OCCUPIED');
        occupiedRooms.sort((a, b) => a.name.localeCompare(b.name, undefined, { numeric: true, sensitivity: 'base' }));

        if (occupiedRooms.length === 0) {
          if (isMounted) {
            setRecommendedRoomId('');
            setIsLoadingRoom(false);
          }
          return;
        }

        const allInvoices = await getInvoices({ house_id: houseId, limit: 1000 });
        const periodInvoices = (allInvoices || []).filter(inv => inv.period === period);
        
        let foundId = '';
        
        for (const room of occupiedRooms) {
          if (!periodInvoices.some(inv => inv.room_id === room.id)) {
            foundId = room.id;
            break;
          }
        }

        // If all rooms have invoices, fallback to the first occupied room
        if (!foundId && occupiedRooms.length > 0) {
            foundId = occupiedRooms[0].id;
        }

        if (isMounted) {
          setRecommendedRoomId(foundId);
        }
      } catch (error) {
        console.error("Error finding recommended room", error);
        if (isMounted) setRecommendedRoomId('');
      } finally {
        if (isMounted) setIsLoadingRoom(false);
      }
    };

    findRoom();

    return () => { isMounted = false; };
  }, [houseId, period, rooms]);

  return { recommendedRoomId, isLoadingRoom };
}

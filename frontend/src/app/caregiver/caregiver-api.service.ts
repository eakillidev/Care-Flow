import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Visit } from '../coordinator/models';

export interface Coordinates { latitude: number; longitude: number; }

@Injectable({ providedIn: 'root' })
export class CaregiverApiService {
  constructor(private readonly http: HttpClient) {}
  listVisits() { return this.http.get<Visit[]>('/api/caregiver/visits'); }
  getVisit(id: string) { return this.http.get<Visit>(`/api/caregiver/visits/${id}`); }
  checkIn(id: string, coordinates: Coordinates) { return this.http.post(`/api/caregiver/visits/${id}/check-in`, coordinates); }
  checkOut(id: string, coordinates: Coordinates) { return this.http.post(`/api/caregiver/visits/${id}/check-out`, coordinates); }
}

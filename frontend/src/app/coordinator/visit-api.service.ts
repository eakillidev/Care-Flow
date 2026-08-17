import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Filters, Patient, PersonSummary, Summary, Visit } from './models';

@Injectable({ providedIn: 'root' })
export class VisitApiService {
  private readonly baseUrl = '/api';
  constructor(private readonly http: HttpClient) {}
  listVisits(filters: Filters) { return this.http.get<Visit[]>(`${this.baseUrl}/visits`, { params: this.params(filters) }); }
  summary(filters: Pick<Filters, 'from' | 'to'>) { return this.http.get<Summary>(`${this.baseUrl}/visits/evv-summary`, { params: this.params(filters) }); }
  getVisit(id: string) { return this.http.get<Visit>(`${this.baseUrl}/visits/${id}`); }
  caregivers() { return this.http.get<PersonSummary[]>(`${this.baseUrl}/caregivers`); }
  patients() { return this.http.get<Patient[]>(`${this.baseUrl}/patients`); }
  private params(filters: Filters | Pick<Filters, 'from' | 'to'>) { let params = new HttpParams(); for (const [key, value] of Object.entries(filters)) if (value) params = params.set(key, value); return params; }
}

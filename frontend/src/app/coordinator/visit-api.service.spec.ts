import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { VisitApiService } from './visit-api.service';

describe('VisitApiService', () => {
  let service: VisitApiService;
  let http: HttpTestingController;
  beforeEach(() => { TestBed.configureTestingModule({ providers: [provideHttpClient(), provideHttpClientTesting()] }); service = TestBed.inject(VisitApiService); http = TestBed.inject(HttpTestingController); });
  afterEach(() => http.verify());
  it('constructs server-side filter query parameters', () => {
    service.listVisits({ status: 'completed', evv_status: 'exception', caregiver_id: 'caregiver', patient_id: 'patient', from: '2026-08-01', to: '2026-08-31' }).subscribe();
    const request = http.expectOne(req => req.url === '/api/visits');
    expect(request.request.params.get('status')).toBe('completed');
    expect(request.request.params.get('evv_status')).toBe('exception');
    expect(request.request.params.get('caregiver_id')).toBe('caregiver');
    expect(request.request.params.get('patient_id')).toBe('patient');
    expect(request.request.params.get('from')).toBe('2026-08-01');
    expect(request.request.params.get('to')).toBe('2026-08-31');
    request.flush([]);
  });
});

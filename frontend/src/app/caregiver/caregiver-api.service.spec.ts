import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { CaregiverApiService } from './caregiver-api.service';

describe('CaregiverApiService', () => {
  let service: CaregiverApiService;
  let http: HttpTestingController;
  beforeEach(() => { TestBed.configureTestingModule({ providers: [provideHttpClient(), provideHttpClientTesting()] }); service = TestBed.inject(CaregiverApiService); http = TestBed.inject(HttpTestingController); });
  afterEach(() => http.verify());
  it('submits only coordinates for check-in', () => {
    service.checkIn('visit-1', { latitude: 39.29, longitude: -76.61 }).subscribe();
    const request = http.expectOne('/api/caregiver/visits/visit-1/check-in');
    expect(request.request.body).toEqual({ latitude: 39.29, longitude: -76.61 });
    expect(request.request.body.caregiver_id).toBeUndefined();
    expect(request.request.body.timestamp).toBeUndefined();
    request.flush({});
  });
});

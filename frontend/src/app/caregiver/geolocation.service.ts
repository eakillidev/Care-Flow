import { Injectable } from '@angular/core';
import { Coordinates } from './caregiver-api.service';

export class LocationError extends Error {
  constructor(readonly code: number) { super('Unable to retrieve location'); }
}

@Injectable({ providedIn: 'root' })
export class GeolocationService {
  current(): Promise<Coordinates> {
    if (!navigator.geolocation) return Promise.reject(new LocationError(0));
    return new Promise((resolve, reject) => navigator.geolocation.getCurrentPosition(
      position => resolve({ latitude: position.coords.latitude, longitude: position.coords.longitude }),
      error => reject(new LocationError(error.code)),
      { enableHighAccuracy: true, timeout: 10_000, maximumAge: 0 }
    ));
  }
}

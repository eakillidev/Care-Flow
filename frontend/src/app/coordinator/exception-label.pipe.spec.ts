import { describe, expect, it } from 'vitest';
import { ExceptionLabelPipe } from './exception-label.pipe';

describe('ExceptionLabelPipe', () => {
  const pipe = new ExceptionLabelPipe();
  it('formats stable EVV reason codes', () => {
    expect(pipe.transform('early_check_in')).toBe('Early check-in');
    expect(pipe.transform('late_check_in')).toBe('Late check-in');
    expect(pipe.transform('outside_geofence')).toBe('Outside allowed location');
    expect(pipe.transform('checkout_outside_geofence')).toBe('Checkout outside allowed location');
  });
});

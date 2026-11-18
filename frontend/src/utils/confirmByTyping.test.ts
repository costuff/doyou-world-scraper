import { describe, expect, it } from 'vitest';
import { confirmByTyping } from './confirmByTyping';

describe('confirmByTyping', () => {
  it('confirms only when the exact phrase is typed', () => {
    expect(confirmByTyping('Delete files?', 'DELETE NAS', () => ' DELETE NAS ')).toBe(true);
    expect(confirmByTyping('Delete files?', 'DELETE NAS', () => 'delete nas')).toBe(false);
    expect(confirmByTyping('Delete files?', 'DELETE NAS', () => '')).toBe(false);
    expect(confirmByTyping('Delete files?', 'DELETE NAS', () => null)).toBe(false);
  });
});

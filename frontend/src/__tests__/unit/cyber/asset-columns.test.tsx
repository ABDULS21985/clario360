import { describe, it, expect } from 'vitest';
import { TYPE_ICONS, TYPE_LABELS } from '@/app/(dashboard)/cyber/assets/_components/asset-columns';

describe('asset-columns constants', () => {
  describe('TYPE_ICONS', () => {
    it('has icons for all asset types', () => {
      const expectedTypes = ['server', 'endpoint', 'cloud_resource', 'network_device', 'iot_device', 'application', 'database', 'container'];
      expectedTypes.forEach((type) => {
        expect(TYPE_ICONS[type as keyof typeof TYPE_ICONS]).toBeDefined();
      });
    });

    it('icons are React components (functions or classes)', () => {
      Object.values(TYPE_ICONS).forEach((Icon) => {
        expect(typeof Icon).toBe('function');
      });
    });
  });

  describe('TYPE_LABELS', () => {
    it('has labels for all asset types', () => {
      const expectedTypes = ['server', 'endpoint', 'cloud_resource', 'network_device', 'iot_device', 'application', 'database', 'container'];
      expectedTypes.forEach((type) => {
        expect(TYPE_LABELS[type as keyof typeof TYPE_LABELS]).toBeDefined();
        expect(typeof TYPE_LABELS[type as keyof typeof TYPE_LABELS]).toBe('string');
        expect(TYPE_LABELS[type as keyof typeof TYPE_LABELS].length).toBeGreaterThan(0);
      });
    });

    it('has correct label for server', () => {
      expect(TYPE_LABELS.server).toBe('Server');
    });

    it('has correct label for database', () => {
      expect(TYPE_LABELS.database).toBe('Database');
    });

    it('has correct label for cloud_resource', () => {
      expect(TYPE_LABELS.cloud_resource).toBe('Cloud');
    });
  });
});

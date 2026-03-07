import type {
  ConnectionTestResult,
  DataSource,
  DiscoveredSchema,
} from '@/lib/data-suite';
import type { SourceConfigureValues, SourceTypeValue } from '@/lib/data-suite/forms';

export interface SourceWizardState {
  step: number;
  sourceType?: SourceTypeValue;
  connectionConfig: Record<string, unknown>;
  createdSource: DataSource | null;
  persistedConfigSignature: string | null;
  testResult: ConnectionTestResult | null;
  testError: string | null;
  schema: DiscoveredSchema | null;
  schemaReviewed: boolean;
  skippedVerificationDetails: boolean;
  configuration: SourceConfigureValues;
}

export function createInitialSourceWizardState(): SourceWizardState {
  return {
    step: 1,
    sourceType: undefined,
    connectionConfig: {},
    createdSource: null,
    persistedConfigSignature: null,
    testResult: null,
    testError: null,
    schema: null,
    schemaReviewed: false,
    skippedVerificationDetails: false,
    configuration: {
      name: '',
      description: '',
      tags: [],
      sync_frequency: null,
    },
  };
}

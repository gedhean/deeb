import React, { useState } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Input } from '@/components/ui/input';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion';
import { useSQLReviewRules } from '@/hooks/api/usePoliciesSWR';
import type {
  DatabaseEngine,
  RuleCategory,
  RuleMetadata,
  RuleParameter,
  SQLReviewPolicyConfig,
  SQLReviewRule,
  SQLReviewRuleLevel,
} from '@/lib/api/types';

interface SQLReviewRuleSelectorProps {
  config: SQLReviewPolicyConfig;
  onChange: (config: SQLReviewPolicyConfig) => void;
}

export const SQLReviewRuleSelector: React.FC<SQLReviewRuleSelectorProps> = ({
  config,
  onChange,
}) => {
  // Use SWR hook for automatic caching and revalidation
  const { rules: rulesResponse, isLoading } = useSQLReviewRules();
  const [selectedEngine, setSelectedEngine] = useState<DatabaseEngine>('mysql');

  const rules: Record<DatabaseEngine, Record<RuleCategory, RuleMetadata[]>> = rulesResponse
    ? {
        mysql: (rulesResponse.mysql || {}) as Record<RuleCategory, RuleMetadata[]>,
        postgres: (rulesResponse.postgres || {}) as Record<RuleCategory, RuleMetadata[]>,
      }
    : ({} as Record<DatabaseEngine, Record<RuleCategory, RuleMetadata[]>>);

  const convertParametersToPayload = (parameters?: RuleParameter[]): Record<string, unknown> => {
    if (!parameters || !Array.isArray(parameters)) {
      return {};
    }

    // Convert array of parameter objects to a map of name -> default value
    const payload: Record<string, unknown> = {};
    for (const param of parameters) {
      if (param.name && param.default !== undefined) {
        payload[param.name] = param.default;
      }
    }
    return payload;
  };

  const handleRuleToggle = (ruleKey: string, ruleMetadata: RuleMetadata, enabled: boolean) => {
    // Deep clone rules to avoid mutating the original config
    const newRules = config.rules ? { ...config.rules } : {};

    if (enabled) {
      newRules[ruleKey] = {
        type: ruleMetadata.type,
        level: 'warning',
        engine: ruleMetadata.engine,
        payload: convertParametersToPayload(ruleMetadata.parameters),
      };
    } else {
      delete newRules[ruleKey];
    }

    onChange({ ...config, rules: newRules });
  };

  const handleLevelChange = (ruleKey: string, level: SQLReviewRuleLevel) => {
    if (!config.rules?.[ruleKey]) return;

    const newRules = {
      ...config.rules,
      [ruleKey]: { ...config.rules[ruleKey], level },
    };
    onChange({ ...config, rules: newRules });
  };

  const handleParameterChange = (ruleKey: string, paramName: string, value: string | number) => {
    if (!config.rules?.[ruleKey]) return;

    const existingRule = config.rules[ruleKey];
    const newPayload = { ...existingRule.payload, [paramName]: value };
    const newRules = {
      ...config.rules,
      [ruleKey]: { ...existingRule, payload: newPayload },
    };
    onChange({ ...config, rules: newRules });
  };

  const isRuleEnabled = (ruleKey: string): boolean => {
    return config.rules?.[ruleKey] !== undefined;
  };

  const getRuleConfig = (ruleKey: string): SQLReviewRule | undefined => {
    return config.rules?.[ruleKey];
  };

  const renderRuleCard = (ruleMetadata: RuleMetadata) => {
    const ruleKey = `${ruleMetadata.engine}.${ruleMetadata.type}`;
    const enabled = isRuleEnabled(ruleKey);
    const ruleConfig = getRuleConfig(ruleKey);

    return (
      <Card key={ruleMetadata.type} className="mb-4">
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <div className="flex-1">
              <CardTitle className="text-sm font-medium">{ruleMetadata.name}</CardTitle>
              <CardDescription className="text-xs mt-1">{ruleMetadata.description}</CardDescription>
            </div>
            <Switch
              checked={enabled}
              onCheckedChange={(checked) => handleRuleToggle(ruleKey, ruleMetadata, checked)}
            />
          </div>
        </CardHeader>
        {enabled && (
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor={`level-${ruleKey}`} className="text-xs">
                Severity Level
              </Label>
              <Select
                value={ruleConfig?.level || 'warning'}
                onValueChange={(value) => handleLevelChange(ruleKey, value as SQLReviewRuleLevel)}
              >
                <SelectTrigger id={`level-${ruleKey}`} className="h-8 text-xs">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="error">Error</SelectItem>
                  <SelectItem value="warning">Warning</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {ruleMetadata.parameters &&
              ruleMetadata.parameters.map((param) => {
                const currentValue = ruleConfig?.payload?.[param.name] ?? param.default;
                const paramType = param.type; // Use the type from parameter definition

                return (
                  <div key={param.name} className="space-y-2">
                    <Label
                      htmlFor={`param-${ruleKey}-${param.name}`}
                      className="text-xs capitalize"
                    >
                      {param.name.replace(/_/g, ' ')}
                    </Label>
                    {paramType === 'NUMBER' ? (
                      <Input
                        id={`param-${ruleKey}-${param.name}`}
                        type="number"
                        value={currentValue as number}
                        onChange={(e) =>
                          handleParameterChange(ruleKey, param.name, Number(e.target.value))
                        }
                        className="h-8 text-xs"
                      />
                    ) : paramType === 'BOOLEAN' ? (
                      <Select
                        value={String(currentValue)}
                        onValueChange={(value) =>
                          handleParameterChange(ruleKey, param.name, String(value === 'true'))
                        }
                      >
                        <SelectTrigger
                          id={`param-${ruleKey}-${param.name}`}
                          className="h-8 text-xs"
                        >
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="true">True</SelectItem>
                          <SelectItem value="false">False</SelectItem>
                        </SelectContent>
                      </Select>
                    ) : (
                      <Input
                        id={`param-${ruleKey}-${param.name}`}
                        type="text"
                        value={String(currentValue)}
                        onChange={(e) => handleParameterChange(ruleKey, param.name, e.target.value)}
                        className="h-8 text-xs"
                      />
                    )}
                  </div>
                );
              })}
          </CardContent>
        )}
      </Card>
    );
  };

  const renderCategory = (category: RuleCategory, categoryRules: RuleMetadata[]) => {
    if (categoryRules.length === 0) {
      return null;
    }

    return (
      <AccordionItem key={category} value={category}>
        <AccordionTrigger className="text-sm font-medium">{category}</AccordionTrigger>
        <AccordionContent>
          <div className="space-y-2">{categoryRules.map((rule) => renderRuleCard(rule))}</div>
        </AccordionContent>
      </AccordionItem>
    );
  };

  if (isLoading) {
    return <div className="text-sm text-muted-foreground">Loading rules...</div>;
  }

  const engineRules = rules[selectedEngine] || ({} as Record<RuleCategory, RuleMetadata[]>);
  const categories = Object.keys(engineRules) as RuleCategory[];

  return (
    <div className="space-y-4">
      <Tabs
        value={selectedEngine}
        onValueChange={(value) => setSelectedEngine(value as DatabaseEngine)}
      >
        <TabsList className="grid w-full grid-cols-2">
          <TabsTrigger value="mysql">MySQL</TabsTrigger>
          <TabsTrigger value="postgres">PostgreSQL</TabsTrigger>
        </TabsList>

        <TabsContent value="mysql" className="mt-4">
          <Accordion type="multiple" className="w-full">
            {categories.map((category) => renderCategory(category, engineRules[category] || []))}
          </Accordion>
        </TabsContent>

        <TabsContent value="postgres" className="mt-4">
          <Accordion type="multiple" className="w-full">
            {categories.map((category) => renderCategory(category, engineRules[category] || []))}
          </Accordion>
        </TabsContent>
      </Tabs>
    </div>
  );
};

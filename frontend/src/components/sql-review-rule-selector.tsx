import React, { useState, useEffect } from 'react';
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
import { policiesService } from '@/lib/api/services/policies';
import type {
  DatabaseType,
  RuleCategory,
  RuleMetadata,
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
  const [rules, setRules] = useState<Record<DatabaseType, Record<RuleCategory, RuleMetadata[]>>>(
    {} as Record<DatabaseType, Record<RuleCategory, RuleMetadata[]>>,
  );
  const [loading, setLoading] = useState(true);
  const [selectedEngine, setSelectedEngine] = useState<DatabaseType>('mysql');

  useEffect(() => {
    const loadRules = async () => {
      try {
        const response = await policiesService.getSQLReviewRules();
        setRules({
          mysql: (response.mysql || {}) as Record<RuleCategory, RuleMetadata[]>,
          postgresql: (response.postgresql || {}) as Record<RuleCategory, RuleMetadata[]>,
        });
      } catch (error) {
        console.error('Failed to load rules:', error);
      } finally {
        setLoading(false);
      }
    };

    loadRules();
  }, []);

  const handleRuleToggle = (ruleKey: string, ruleMetadata: RuleMetadata, enabled: boolean) => {
    const newConfig = { ...config };
    if (!newConfig.rules) {
      newConfig.rules = {};
    }

    if (enabled) {
      newConfig.rules[ruleKey] = {
        type: ruleMetadata.type,
        level: 'warning',
        engine: ruleMetadata.engine,
        payload: ruleMetadata.parameters ? { ...ruleMetadata.parameters } : {},
      };
    } else {
      delete newConfig.rules[ruleKey];
    }

    onChange(newConfig);
  };

  const handleLevelChange = (ruleKey: string, level: SQLReviewRuleLevel) => {
    const newConfig = { ...config };
    if (newConfig.rules[ruleKey]) {
      newConfig.rules[ruleKey].level = level;
      onChange(newConfig);
    }
  };

  const handleParameterChange = (ruleKey: string, paramName: string, value: string | number) => {
    const newConfig = { ...config };
    if (newConfig.rules[ruleKey]) {
      if (!newConfig.rules[ruleKey].payload) {
        newConfig.rules[ruleKey].payload = {};
      }
      newConfig.rules[ruleKey].payload![paramName] = value;
      onChange(newConfig);
    }
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
              Object.entries(ruleMetadata.parameters).map(([paramName, paramValue]) => {
                const paramType = typeof paramValue;
                const currentValue = ruleConfig?.payload?.[paramName] ?? paramValue;

                return (
                  <div key={paramName} className="space-y-2">
                    <Label htmlFor={`param-${ruleKey}-${paramName}`} className="text-xs capitalize">
                      {paramName.replace(/_/g, ' ')}
                    </Label>
                    {paramType === 'number' ? (
                      <Input
                        id={`param-${ruleKey}-${paramName}`}
                        type="number"
                        value={currentValue as number}
                        onChange={(e) =>
                          handleParameterChange(ruleKey, paramName, Number(e.target.value))
                        }
                        className="h-8 text-xs"
                      />
                    ) : (
                      <Input
                        id={`param-${ruleKey}-${paramName}`}
                        type="text"
                        value={String(currentValue)}
                        onChange={(e) => handleParameterChange(ruleKey, paramName, e.target.value)}
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

  if (loading) {
    return <div className="text-sm text-muted-foreground">Loading rules...</div>;
  }

  const engineRules = rules[selectedEngine] || ({} as Record<RuleCategory, RuleMetadata[]>);
  const categories = Object.keys(engineRules) as RuleCategory[];

  return (
    <div className="space-y-4">
      <Tabs
        value={selectedEngine}
        onValueChange={(value) => setSelectedEngine(value as DatabaseType)}
      >
        <TabsList className="grid w-full grid-cols-2">
          <TabsTrigger value="mysql">MySQL</TabsTrigger>
          <TabsTrigger value="postgresql">PostgreSQL</TabsTrigger>
        </TabsList>

        <TabsContent value="mysql" className="mt-4">
          <Accordion type="multiple" className="w-full">
            {categories.map((category) => renderCategory(category, engineRules[category] || []))}
          </Accordion>
        </TabsContent>

        <TabsContent value="postgresql" className="mt-4">
          <Accordion type="multiple" className="w-full">
            {categories.map((category) => renderCategory(category, engineRules[category] || []))}
          </Accordion>
        </TabsContent>
      </Tabs>
    </div>
  );
};

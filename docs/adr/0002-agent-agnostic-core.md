# ADR-002 — Agent-agnostic Core

- Status: Accepted
- Date: 2026-10-01

## Context

Binding WOS state to a particular agent runtime, Session or Run would prevent human-only use and make continuation depend on the infrastructure that initiated work.

## Decision

The Core is independent from Woobe, LLMs and agent runtimes. ActorRef is generic attribution. Runtime/session information is optional execution context handled outside the domain identity model.

## Consequences

WOS can be used standalone, embedded or through protocol adapters. Woobe and other systems integrate by contract instead of becoming Core dependencies.

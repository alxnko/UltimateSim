# Phase 33: Advanced Medicine, Surgery & Mutilation

## Overview
Combat and accidents should have permanent, physical consequences. Health is not just a single HP bar; it is a complex system of limbs, organs, infections, and medieval medicine.

## Architecture & DOD Implementation
- **`AnatomyComponent`**: 
  - A complex DOD struct tracking the status of individual body parts (Head, Torso, L. Arm, R. Arm, L. Leg, R. Leg, Internal Organs).
  - Damage is localized. A crushed leg drastically reduces `Velocity`. A severed arm prevents the use of two-handed weapons or certain `JobContracts` (like Blacksmithing).
- **Infection & Triage**: 
  - Open wounds have a high probability of generating a localized `InfectionMarker` based on the map's `BiomeID` (e.g., swamps are highly infectious).
  - Infections slowly drain `Vitals.Blood` and increase `Vitals.Pain` unless treated.
- **Medical Labor & Surgery**: 
  - NPCs with the `JobDoctor` tag can accept `MedicalContracts`.
  - Treatment requires physical items: Bandages (crafted from cloth), Herbs (Phase 30), or Alcohol (for pain). 
  - In extreme cases of infection, doctors can perform amputations to save the entity's life, permanently altering the `AnatomyComponent`.
- **Prosthetics (Peg Legs & Hook Hands)**: 
  - The crafting system (Phase 26) can produce physical prosthetics from wood or iron, which can be surgically attached to restore partial functionality to mutilated entities.
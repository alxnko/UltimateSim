import sys

def fix_labor_union():
    with open('internal/systems/labor_union.go', 'r') as f:
        content = f.read()

    search = """		// 3. O(N) evaluation against the flat active strikes cache
		for _, strike := range activeStrikes {
			if job.EmployerID == strike.TargetEmployerID {
				// The NPC is actively working for a business currently being struck.
				// Phase 24.1: The Butterfly Effect (Labor, Economy, Justice)
				// Striker generates massive negative hooks (-50) against the scab, triggering BloodFeudSystem (Phase 23.1)
				s.hookGraph.AddHook(strike.StrikerID, scabID.ID, -50)

				// Striker generates moderate negative hooks (-10) against the offending employer
				s.hookGraph.AddHook(strike.StrikerID, job.EmployerID, -10)
			}
		}"""

    replace = """		// 3. O(N) evaluation against the flat active strikes cache
		for _, strike := range activeStrikes {
			if job.EmployerID == strike.TargetEmployerID {
				// Check if the hook already exists to prevent massive infinite stack overflow per tick.
				// We want a static -50 baseline.
				currentScabHook := s.hookGraph.GetHook(strike.StrikerID, scabID.ID)
				if currentScabHook > -50 {
					// The NPC is actively working for a business currently being struck.
					// Phase 24.1: The Butterfly Effect (Labor, Economy, Justice)
					// Striker generates massive negative hooks (-50) against the scab, triggering BloodFeudSystem (Phase 23.1)
					s.hookGraph.AddHook(strike.StrikerID, scabID.ID, -50)
				}

				currentEmpHook := s.hookGraph.GetHook(strike.StrikerID, job.EmployerID)
				if currentEmpHook > -10 {
					// Striker generates moderate negative hooks (-10) against the offending employer
					s.hookGraph.AddHook(strike.StrikerID, job.EmployerID, -10)
				}
			}
		}"""

    if search in content:
        content = content.replace(search, replace)
    else:
        print("Search string not found in labor_union.go")

    with open('internal/systems/labor_union.go', 'w') as f:
        f.write(content)

if __name__ == "__main__":
    fix_labor_union()
    print("Fixed labor_union.go")

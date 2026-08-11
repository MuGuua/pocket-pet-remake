import { Spin } from 'antd';
import { useEffect, useState } from 'react';
import { Navigate, Route, Routes, useLocation } from 'react-router-dom';
import { getAdminToken } from '../services/http';
import { fetchAdminProfile, logoutAdmin } from '../services/auth';
import type { AdminSessionProfile } from '../types/admin';
import { AdminLayout } from '../layouts/AdminLayout';
import { LoginPage } from '../pages/auth/LoginPage';
import { DashboardPage } from '../pages/dashboard/DashboardPage';
import { QuestAdminPage } from '../pages/quests/QuestAdminPage';
import { NPCConfigPage } from '../pages/npcs/NPCConfigPage';
import { ItemTemplatePage } from '../pages/items/ItemTemplatePage';
import { PetManagementPage } from '../pages/pets/PetManagementPage';
import { SkillDefinitionPage } from '../pages/skills/SkillDefinitionPage';
import { MonsterDefinitionPage } from '../pages/monsters/MonsterDefinitionPage';
import { EncounterConfigPage } from '../pages/monsters/EncounterConfigPage';
import { EquipmentDefinitionPage } from '../pages/equipment/EquipmentDefinitionPage';
import { PlayerManagementPage } from '../pages/players/PlayerManagementPage';
import { WorldMovementConfigPage } from '../pages/world/WorldMovementConfigPage';

function RequireAdminAuth() {
  const location = useLocation();
  const [loading, setLoading] = useState(true);
  const [profile, setProfile] = useState<AdminSessionProfile | null>(null);

  useEffect(() => {
    let cancelled = false;
    const token = getAdminToken();
    if (!token) {
      setLoading(false);
      setProfile(null);
      return;
    }

    fetchAdminProfile()
      .then((nextProfile) => {
        if (!cancelled) {
          setProfile(nextProfile);
        }
      })
      .catch(() => {
        if (!cancelled) {
          logoutAdmin();
          setProfile(null);
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [location.pathname]);

  if (loading) {
    return (
      <div style={{ minHeight: '100vh', display: 'grid', placeItems: 'center' }}>
        <Spin size="large" tip="正在校验后台登录态..." />
      </div>
    );
  }

  if (!profile) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />;
  }

  return <AdminLayout profile={profile} />;
}

export function AppRouter() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path="/" element={<RequireAdminAuth />}>
        <Route index element={<Navigate to="/dashboard" replace />} />
        <Route path="dashboard" element={<DashboardPage />} />
        <Route path="players" element={<PlayerManagementPage />} />
        <Route path="player-progression" element={<Navigate to="/players?tab=progression" replace />} />
        <Route path="pets" element={<PetManagementPage />} />
        <Route path="pet-progression" element={<Navigate to="/pets?tab=progression" replace />} />
        <Route path="pet-skill-slot-unlock" element={<Navigate to="/pets?tab=skill-slots" replace />} />
        <Route path="player-pets" element={<Navigate to="/players" replace />} />
        <Route path="pet-combat-stat-caps" element={<Navigate to="/pets?tab=combat-caps" replace />} />
        <Route path="equipment-definitions" element={<Navigate to="/items?tab=equipment" replace />} />
        <Route path="pet-definitions" element={<Navigate to="/pets?tab=definitions" replace />} />
        <Route path="skill-definitions" element={<SkillDefinitionPage />} />
        <Route path="monster-definitions" element={<MonsterDefinitionPage />} />
        <Route path="monster-encounters" element={<EncounterConfigPage />} />
        <Route path="scene-wild-encounters" element={<Navigate to="/monster-encounters" replace />} />
        <Route path="items" element={<ItemTemplatePage />} />
        <Route path="quests" element={<QuestAdminPage />} />
        <Route path="npcs" element={<NPCConfigPage />} />
        <Route path="world-movement" element={<WorldMovementConfigPage />} />
        <Route path="npc-dialogues" element={<Navigate to="/npcs" replace />} />
      </Route>
    </Routes>
  );
}

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
import { ItemDefinitionPage } from '../pages/items/ItemDefinitionPage';
import { PetDefinitionPage } from '../pages/pets/PetDefinitionPage';
import { SkillDefinitionPage } from '../pages/skills/SkillDefinitionPage';
import { MonsterDefinitionPage } from '../pages/monsters/MonsterDefinitionPage';
import { EncounterConfigPage } from '../pages/monsters/EncounterConfigPage';
import { PlayerProgressionPage } from '../pages/progression/PlayerProgressionPage';
import { PlayerListPage } from '../pages/players/PlayerListPage';

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
        <Route path="players" element={<PlayerListPage />} />
        <Route path="player-progression" element={<PlayerProgressionPage />} />
        <Route path="pet-definitions" element={<PetDefinitionPage />} />
        <Route path="skill-definitions" element={<SkillDefinitionPage />} />
        <Route path="monster-definitions" element={<MonsterDefinitionPage />} />
        <Route path="monster-encounters" element={<EncounterConfigPage />} />
        <Route path="scene-wild-encounters" element={<Navigate to="/monster-encounters" replace />} />
        <Route path="items" element={<ItemDefinitionPage />} />
        <Route path="quests" element={<QuestAdminPage />} />
        <Route path="npcs" element={<NPCConfigPage />} />
      </Route>
    </Routes>
  );
}
